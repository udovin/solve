package api

import (
	"database/sql"
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/udovin/solve/managers"
	"github.com/udovin/solve/models"
)

func (v *View) registerContestMessageHandlers(g *echo.Group) {
	g.GET(
		"/v0/contests/:contest/messages", v.observeContestMessages,
		v.extractAuth(v.sessionAuth, v.guestAuth), v.extractContest,
		v.requirePermission(models.ObserveContestMessagesRole),
	)
	g.POST(
		"/v0/contests/:contest/messages", v.createContestMessage,
		v.extractAuth(v.sessionAuth), v.extractContest,
		v.requirePermission(models.CreateContestMessageRole),
	)
	g.POST(
		"/v0/contests/:contest/submit-question", v.submitContestQuestion,
		v.extractAuth(v.sessionAuth), v.extractContest,
		v.requirePermission(models.SubmitContestQuestionRole),
	)
}

func (v *View) observeContestMessages(c echo.Context) error {
	contestCtx, ok := c.Get(contestCtxKey).(*managers.ContestContext)
	if !ok {
		return fmt.Errorf("contest not extracted")
	}
	if err := syncStore(c, v.core.ContestMessages); err != nil {
		return err
	}
	messages, err := v.core.ContestMessages.FindByContest(
		getContext(c), contestCtx.Contest.ID,
	)
	if err != nil {
		return err
	}
	defer func() { _ = messages.Close() }()
	var resp ContestMessages
	for messages.Next() {
		message := messages.Row()
		permissions := v.getContestMessagePermissions(contestCtx, message)
		if permissions.HasPermission(models.ObserveContestMessageRole) {
			resp.Messages = append(
				resp.Messages,
				makeContestMessage(message),
			)
		}
	}
	if err := messages.Err(); err != nil {
		return err
	}
	sortFunc(resp.Messages, contestMessageGreater)
	return c.JSON(http.StatusOK, resp)
}

type CreateContestMessageForm struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	ParentID    *int64 `json:"parent_id"`
}

func (f *CreateContestMessageForm) Update(
	c echo.Context, o *models.ContestMessage,
	messages models.ContestMessageStore,
) error {
	errors := errorFields{}
	if f.ParentID == nil {
		if len(f.Title) < 4 {
			errors["title"] = errorField{
				Message: localize(c, "Title is too short."),
			}
		} else if len(f.Title) > 64 {
			errors["title"] = errorField{
				Message: localize(c, "Title is too long."),
			}
		}
	} else {
		f.Title = ""
	}
	if len(f.Description) < 4 {
		errors["description"] = errorField{
			Message: localize(c, "Description is too short."),
		}
	} else if len(f.Description) > 1024 {
		errors["description"] = errorField{
			Message: localize(c, "Description is too long."),
		}
	}
	if len(errors) > 0 {
		return errorResponse{
			Code:          http.StatusBadRequest,
			Message:       localize(c, "Form has invalid fields."),
			InvalidFields: errors,
		}
	}
	if f.ParentID != nil {
		message, err := messages.Get(getContext(c), *f.ParentID)
		if err != nil {
			if err != sql.ErrNoRows {
				return err
			}
			return errorResponse{
				Code:    http.StatusBadRequest,
				Message: localize(c, "Message not found."),
			}
		}
		if message.Kind != models.QuestionContestMessage {
			return errorResponse{
				Code:    http.StatusBadRequest,
				Message: localize(c, "Message should be a question."),
			}
		}
		o.Kind = models.AnswerContestMessage
		o.ParentID = models.NInt64(*f.ParentID)
	} else {
		o.Kind = models.RegularContestMessage
	}
	o.Title = f.Title
	o.Description = f.Description
	return nil
}

func (v *View) createContestMessage(c echo.Context) error {
	contestCtx, ok := c.Get(contestCtxKey).(*managers.ContestContext)
	if !ok {
		return fmt.Errorf("contest not extracted")
	}
	var form CreateContestMessageForm
	if err := c.Bind(&form); err != nil {
		return err
	}
	now := getNow(c)
	message := models.ContestMessage{
		ContestID:  contestCtx.Contest.ID,
		AuthorID:   contestCtx.Account.ID,
		CreateTime: now.Unix(),
	}
	if err := form.Update(c, &message, v.core.ContestMessages); err != nil {
		return err
	}
	if err := v.core.ContestMessages.Create(getContext(c), &message); err != nil {
		return err
	}
	return c.JSON(http.StatusCreated, makeContestMessage(message))
}

type SubmitContestQuestionForm struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

func (f SubmitContestQuestionForm) Update(
	c echo.Context, o *models.ContestMessage,
) error {
	errors := errorFields{}
	if len(f.Title) < 4 {
		errors["title"] = errorField{
			Message: localize(c, "Title is too short."),
		}
	} else if len(f.Title) > 64 {
		errors["title"] = errorField{
			Message: localize(c, "Title is too long."),
		}
	}
	if len(f.Description) < 4 {
		errors["description"] = errorField{
			Message: localize(c, "Description is too short."),
		}
	} else if len(f.Description) > 1024 {
		errors["description"] = errorField{
			Message: localize(c, "Description is too long."),
		}
	}
	if len(errors) > 0 {
		return &errorResponse{
			Code:          http.StatusBadRequest,
			Message:       localize(c, "Form has invalid fields."),
			InvalidFields: errors,
		}
	}
	o.Title = f.Title
	o.Description = f.Description
	return nil
}

func (v *View) submitContestQuestion(c echo.Context) error {
	contestCtx, ok := c.Get(contestCtxKey).(*managers.ContestContext)
	if !ok {
		return fmt.Errorf("contest not extracted")
	}
	var form SubmitContestQuestionForm
	if err := c.Bind(&form); err != nil {
		c.Logger().Warn(err)
		return c.NoContent(http.StatusBadRequest)
	}
	participant := contestCtx.GetEffectiveParticipant()
	if participant == nil {
		return errorResponse{
			Code:    http.StatusForbidden,
			Message: localize(c, "Participant not found."),
		}
	}
	if !contestCtx.HasEffectivePermission(models.SubmitContestQuestionRole) {
		return errorResponse{
			Code:               http.StatusForbidden,
			Message:            localize(c, "Account missing permissions."),
			MissingPermissions: []string{models.SubmitContestQuestionRole},
		}
	}
	if participant.ID == 0 {
		if err := v.core.ContestParticipants.Create(
			getContext(c), participant,
		); err != nil {
			return err
		}
	}
	if participant.ID == 0 {
		return fmt.Errorf("unable to register participant")
	}
	message := models.ContestMessage{
		ContestID:     contestCtx.Contest.ID,
		ParticipantID: models.NInt64(participant.ID),
		AuthorID:      contestCtx.Account.ID,
		Kind:          models.QuestionContestMessage,
		CreateTime:    contestCtx.Now.Unix(),
	}
	if err := form.Update(c, &message); err != nil {
		return err
	}
	if err := v.core.ContestMessages.Create(
		getContext(c), &message,
	); err != nil {
		return err
	}
	return c.JSON(http.StatusCreated, makeContestMessage(message))
}

type ContestMessage struct {
	ID          int64  `json:"id"`
	ParentID    int64  `json:"parent_id,omitempty"`
	Kind        string `json:"kind"`
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
}

type ContestMessages struct {
	Messages []ContestMessage `json:"messages"`
}

func makeContestMessage(message models.ContestMessage) ContestMessage {
	resp := ContestMessage{
		ID:          message.ID,
		ParentID:    int64(message.ParentID),
		Kind:        message.Kind.String(),
		Title:       message.Title,
		Description: message.Description,
	}
	return resp
}

func (v *View) getContestMessagePermissions(
	ctx *managers.ContestContext, message models.ContestMessage,
) managers.PermissionSet {
	permissions := ctx.Permissions.Clone()
	if message.ParticipantID != 0 {
		for _, participant := range ctx.Participants {
			if participant.ID == int64(message.ParticipantID) {
				permissions.AddPermission(models.ObserveContestMessageRole)
			}
		}
	} else {
		permissions.AddPermission(models.ObserveContestMessageRole)
	}
	return permissions
}

func contestMessageGreater(l, r ContestMessage) bool {
	return l.ID > r.ID
}
