import React, {useState} from "react";
import Page from "../layout/Page";
import Input from "../layout/Input";
import Button from "../layout/Button";
import {FormBlock} from "../layout/blocks";
import {Problem} from "../api";
import {Redirect, RouteComponentProps} from "react-router";
import Field from "../layout/Field";

type UpdateProblemPageParams = {
	ProblemID: string;
}

const UpdateProblemPage = ({match}: RouteComponentProps<UpdateProblemPageParams>) => {
	const {ProblemID} = match.params;
	const [problem, setProblem] = useState<Problem>();
	const onSubmit = (event: any) => {
		event.preventDefault();
		const {title, file} = event.target;
		let form = new FormData();
		form.append("ID", ProblemID);
		form.append("Title", title.value);
		form.append("File", file.files[0]);
		fetch("/api/v0/problems/" + ProblemID, {
			method: "PATCH",
			body: form,
		})
			.then(result => result.json())
			.then(result => setProblem(result))
			.catch(error => console.log(error));
	};
	if (problem) {
		return <Redirect to={"/problems/" + problem.ID}/>
	}
	return <Page title="Update problem">
		<FormBlock onSubmit={onSubmit} title="Update problem" footer={
			<Button type="submit" color="primary">Update</Button>
		}>
			<Field title="Title:">
				<Input type="text" name="title" placeholder="Title" required autoFocus/>
			</Field>
			<Field title="Package:">
				<Input type="file" name="file" placeholder="Package" required/>
			</Field>
		</FormBlock>
	</Page>;
};

export default UpdateProblemPage;
