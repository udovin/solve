import React from "react";
import Page from "../layout/Page";
import Input from "../layout/Input";
import {Button} from "../layout/buttons";
import {FormBlock} from "../layout/blocks";

const CreateCompilerPage = () => {
	const onSubmit = (event: any) => {
		event.preventDefault();
		const {name} = event.target;
		fetch("/api/v0/compilers", {
			method: "POST",
			headers: {
				"Content-Type": "application/json; charset=UTF-8",
			},
			body: JSON.stringify({
				Name: name.value,
			})
		}).then();
	};
	return <Page title="Create compiler">
		<FormBlock onSubmit={onSubmit} title="Create compiler" footer={
			<Button type="submit" color="primary">Create</Button>
		}>
			<div className="ui-field">
				<label>
					<span className="label">Name:</span>
					<Input type="text" name="name" placeholder="Name" required autoFocus/>
				</label>
			</div>
		</FormBlock>
	</Page>;
};

export default CreateCompilerPage;
