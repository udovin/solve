import React, {useState} from "react";
import Page from "../layout/Page";
import Input from "../layout/Input";
import Button from "../layout/Button";
import {FormBlock} from "../layout/blocks";
import {Contest} from "../api";
import {Redirect} from "react-router";

const CreateContestPage = () => {
	let [contest, setContest] = useState<Contest>();
	const onSubmit = (event: any) => {
		event.preventDefault();
		const {title} = event.target;
		fetch("/api/v0/contests", {
			method: "POST",
			headers: {
				"Content-Type": "application/json; charset=UTF-8",
			},
			body: JSON.stringify({
				Title: title.value,
			})
		})
			.then(result => result.json())
			.then(result => setContest(result));
	};
	if (contest) {
		return <Redirect to={"/contests/" + contest.ID}/>
	}
	return <Page title="Create contest">
		<FormBlock onSubmit={onSubmit} title="Create contest" footer={
			<Button type="submit" color="primary">Create</Button>
		}>
			<div className="ui-field">
				<label>
					<span className="label">Title:</span>
					<Input type="text" name="title" placeholder="Title" required autoFocus/>
				</label>
			</div>
		</FormBlock>
	</Page>;
};

export default CreateContestPage;
