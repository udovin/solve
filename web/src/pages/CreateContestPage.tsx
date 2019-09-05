import React from "react";
import Page from "../layout/Page";
import Input from "../layout/Input";
import {Button} from "../layout/buttons";

const CreateContestPage = () => {
	let handleSubmit = function (event: any) {
		const title = event.target.title.value;
		let request = new XMLHttpRequest();
		request.open("POST", "/api/v0/contests");
		request.setRequestHeader("Content-Type", "application/json; charset=UTF-8");
		request.send(JSON.stringify({
			"Title": title,
		}));
		event.preventDefault();
	};
	return (
		<Page title="Create contest">
			<div className="ui-block-wrap">
				<form className="ui-block" onSubmit={handleSubmit}>
					<div className="ui-block-header">
						<h2 className="title">Create contest</h2>
					</div>
					<div className="ui-block-content">
						<div className="ui-field">
							<label>
								<span className="label">Title:</span>
								<Input
									type="text" name="title"
									placeholder="Title" required
									autoFocus
								/>
							</label>
						</div>
					</div>
					<div className="ui-block-footer">
						<Button type="submit" color="primary">
							Create
						</Button>
					</div>
				</form>
			</div>
		</Page>
	);
};

export default CreateContestPage;
