import React, {useState} from "react";
import Page from "../components/Page";
import Input from "../components/Input";
import Button from "../components/Button";
import FormBlock from "../components/FormBlock";
import {Problem} from "../api";
import {Redirect} from "react-router";

const CreateProblemPage = () => {
	const [problem, setProblem] = useState<Problem>();
	const onSubmit = (event: any) => {
		event.preventDefault();
		const {title, file} = event.target;
		const form = new FormData();
		form.append("Title", title.value);
		form.append("File", file.files[0]);
		fetch("/api/v0/problems", {
			method: "POST",
			body: form,
		})
			.then(result => result.json())
			.then(result => setProblem(result))
			.catch(error => console.log(error));
	};
	if (problem) {
		return <Redirect to={"/problems/" + problem.ID}/>
	}
	return <Page title="Create problem">
		<FormBlock onSubmit={onSubmit} title="Create problem" footer={
			<Button type="submit" color="primary">Create</Button>
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

export default CreateProblemPage;
