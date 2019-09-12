import React, {useState} from "react";
import Page from "../layout/Page";
import Input from "../layout/Input";
import {Button} from "../layout/buttons";
import {FormBlock} from "../layout/blocks";
import {Problem} from "../api";
import {Redirect, RouteComponentProps} from "react-router";

type UpdateProblemPageParams = {
	ProblemID: string;
}

const UpdateProblemPage = ({match}: RouteComponentProps<UpdateProblemPageParams>) => {
	const {ProblemID} = match.params;
	let [problem, setProblem] = useState<Problem>();
	let onSubmit = (event: any) => {
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
			<div className="ui-field">
				<label>
					<span className="label">Title:</span>
					<Input type="text" name="title" placeholder="Title" required autoFocus/>
				</label>
			</div>
			<div className="ui-field">
				<label>
					<span className="label">Package:</span>
					<Input type="file" name="file" placeholder="Package" required/>
				</label>
			</div>
		</FormBlock>
	</Page>;
};

export default UpdateProblemPage;
