import React from "react";
import Page from "../layout/Page";
import Input from "../layout/Input";
import {Button} from "../layout/buttons";
import {FormBlock} from "../layout/blocks";

const CreateProblemPage = () => {
	let onSubmit = (event: any) => {
		event.preventDefault();
		const {title, description} = event.target;
		fetch("/api/v0/problems", {
			method: "POST",
			headers: {
				"Content-Type": "application/json; charset=UTF-8",
			},
			body: JSON.stringify({
				Title: title.value,
				Description: description.value,
			})
		}).then();
	};
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
					<span className="label">Description:</span>
					<Input type="text" name="description" placeholder="Description"/>
				</label>
			</div>
		</FormBlock>
	</Page>;
};

export default CreateProblemPage;
