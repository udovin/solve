import React, {useState} from "react";
import Page from "../layout/Page";
import Input from "../layout/Input";
import Button from "../layout/Button";
import Field from "../layout/Field";
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
		})
			.then(() => setSuccess(true));
	};
	if (success) {
		return <Redirect to={"/contests/" + ContestID}/>
	}
	return <Page title="Add contest problem">
		<FormBlock onSubmit={onSubmit} title="Add contest problem" footer={
			<Button type="submit" color="primary">Create</Button>
		}>
			<Field title="Problem ID:">
				<Input type="number" name="problemID" placeholder="ID" required autoFocus/>
			</Field>
			<Field title="Code:">
				<Input type="text" name="code" placeholder="Code" required/>
			</Field>
		</FormBlock>
	</Page>;
};

export default CreateContestProblemPage;
