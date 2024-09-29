package api

import (
	"context"
	"testing"

	"github.com/udovin/solve/internal/models"
)

func TestGroupSimpleScenario(t *testing.T) {
	e := NewTestEnv(t)
	defer e.Close()
	// Owner.
	user := NewTestUser(e)
	user.AddRoles("observe_groups", "create_group")
	// Regular.
	user1 := NewTestUser(e)
	user1.AddRoles("observe_groups")
	// Manager.
	user2 := NewTestUser(e)
	user2.AddRoles("observe_groups")
	// Non member.
	user3 := NewTestUser(e)
	user3.AddRoles("observe_groups")
	var groupID int64
	func() {
		user.LoginClient()
		defer user.LogoutClient()
		e.Check("Group owner")
		form := CreateGroupForm{
			Title: getPtr("Test group"),
		}
		group, err := e.Client.CreateGroup(context.Background(), form)
		if err != nil {
			t.Fatal("Error:", err)
		} else {
			e.Check(group)
		}
		groupID = group.ID
		if groups, err := e.Client.ObserveGroups(context.Background()); err != nil {
			t.Fatal("Error:", err)
		} else {
			e.Check(groups)
		}
		if v, err := e.Client.ObserveGroup(context.Background(), group.ID); err != nil {
			t.Fatal("Error:", err)
		} else {
			e.Check(v)
		}
		if v, err := e.Client.ObserveGroupMembers(context.Background(), group.ID); err != nil {
			t.Fatal("Error:", err)
		} else {
			e.Check(v)
		}
		{
			form := CreateGroupMemberForm{Kind: models.RegularMember, AccountID: user1.ID}
			member, err := e.Client.CreateGroupMember(context.Background(), group.ID, form)
			if err != nil {
				t.Fatal("Error:", err)
			} else {
				e.Check(member)
			}
		}
		{
			form := CreateGroupMemberForm{Kind: models.RegularMember, AccountID: user2.ID}
			member, err := e.Client.CreateGroupMember(context.Background(), group.ID, form)
			if err != nil {
				t.Fatal("Error:", err)
			} else {
				e.Check(member)
			}
			updateForm := UpdateGroupMemberForm{Kind: models.ManagerMember}
			if v, err := e.Client.UpdateGroupMember(context.Background(), group.ID, member.ID, updateForm); err != nil {
				t.Fatal("Error:", err)
			} else {
				e.Check(v)
			}
		}
		if v, err := e.Client.ObserveGroupMembers(context.Background(), group.ID); err != nil {
			t.Fatal("Error:", err)
		} else {
			e.Check(v)
		}
	}()
	func() {
		user3.LoginClient()
		defer user3.LogoutClient()
		e.Check("Group non member")
		if groups, err := e.Client.ObserveGroups(context.Background()); err != nil {
			t.Fatal("Error:", err)
		} else {
			e.Check(groups)
		}
		if _, err := e.Client.ObserveGroup(context.Background(), groupID); err == nil {
			t.Fatal("Expected error")
		} else {
			e.Check(err)
		}
	}()
	func() {
		user1.LoginClient()
		defer user1.LogoutClient()
		e.Check("Group regular member")
		if groups, err := e.Client.ObserveGroups(context.Background()); err != nil {
			t.Fatal("Error:", err)
		} else {
			e.Check(groups)
		}
		if members, err := e.Client.ObserveGroupMembers(context.Background(), groupID); err != nil {
			t.Fatal("Error:", err)
		} else {
			e.Check(members)
		}
	}()
	func() {
		user2.LoginClient()
		defer user2.LogoutClient()
		e.Check("Group manager member")
		if groups, err := e.Client.ObserveGroups(context.Background()); err != nil {
			t.Fatal("Error:", err)
		} else {
			e.Check(groups)
		}
		if members, err := e.Client.ObserveGroupMembers(context.Background(), groupID); err != nil {
			t.Fatal("Error:", err)
		} else {
			e.Check(members)
		}
		form := CreateGroupMemberForm{Kind: models.RegularMember, AccountID: user3.ID}
		member, err := e.Client.CreateGroupMember(context.Background(), groupID, form)
		if err != nil {
			t.Fatal("Error:", err)
		} else {
			e.Check(member)
		}
		if v, err := e.Client.DeleteGroupMember(context.Background(), groupID, member.ID); err != nil {
			t.Fatal("Error:", err)
		} else {
			e.Check(v)
		}
	}()
}
