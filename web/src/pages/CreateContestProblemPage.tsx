import React, {useState} from "react";
import Page from "../layout/Page";
import Input from "../layout/Input";
import {Button} from "../layout/buttons";
import {FormBlock} from "../layout/blocks";
import {Redirect, RouteComponentProps} from "react-router";

type ContestPageParams = {
	ContestID: string;
}

const CreateContestProblemPage = ({match}: RouteComponentProps<ContestPageParams>) => {
	const {ContestID} = match.params;
	const [success, setSuccess] = useState<boolean>();
	const onSubmit = (event: any) => {
		event.preventDefault();
		const {problemID, code} = event.target;
		fetch("/api/v0/contests/" + ContestID + "/problems", {
			method: "POST",
			headers: {
				"Content-Type": "application/json; charset=UTF-8",
			},
			body: JSON.stringify({
				ProblemID: Number(problemID.value),
				Code: code.value,
			})
		}).then(() => setSuccess(true));
	};
	if (success) {
		return <Redirect to={"/contests/" + ContestID}/>
	}
	return <Page title="Add contest problem">
		<FormBlock onSubmit={onSubmit} title="Add contest problem" footer={
			<Button type="submit" color="primary">Create</Button>
		}>
			<div className="ui-field">
				<label>
					<span className="label">Problem ID:</span>
					<Input type="number" name="problemID" placeholder="ID" required autoFocus/>
				</label>
			</div>
			<div className="ui-field">
				<label>
					<span className="label">Code:</span>
					<Input type="text" name="code" placeholder="Code" required/>
				</label>
			</div>
		</FormBlock>
	</Page>;
};

export default CreateContestProblemPage;
