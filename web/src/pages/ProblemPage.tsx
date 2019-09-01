import React from "react";
import Page from "../layout/Page";
import {RouteComponentProps} from "react-router";
import NotFoundPage from "./NotFoundPage";

type ProblemPageParams = {
	ProblemID: string;
}

type Statement = {
	ID: number;
	ProblemID: number;
	Title: string;
	Description: string;
	CreateTime: number;
}

type Problem = {
	ID: number;
	UserID: number;
	CreateTime: number;
	Statement: Statement;
}

const ProblemPage = ({match}: RouteComponentProps<ProblemPageParams>) => {
	let request = new XMLHttpRequest();
	request.open("GET", "/api/v0/problems/" + match.params.ProblemID, false);
	request.setRequestHeader("Content-Type", "application/json; charset=UTF-8");
	request.send();
	if (request.status !== 200) {
		return <NotFoundPage/>;
	}
	let problem: Problem = JSON.parse(request.response);
	return (
		<Page title={problem.Statement.Title}>
			<div className="ui-block-wrap">
				<div className="ui-block">
					<div className="ui-block-header">
						<h2 className="title">{problem.Statement.Title}</h2>
					</div>
					<div className="ui-block-content">
						{problem.Statement.Description}
					</div>
				</div>
			</div>
		</Page>
	);
};

export default ProblemPage;
