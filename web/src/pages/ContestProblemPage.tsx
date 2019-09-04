import React from "react";
import {RouteComponentProps} from "react-router";
import Page from "../layout/Page";
import NotFoundPage from "./NotFoundPage";
import {ContestProblem} from "../api";

type ContestProblemPageParams = {
	ContestID: string;
	ProblemCode: string;
}

const ContestProblemPage = ({match}: RouteComponentProps<ContestProblemPageParams>) => {
	let request = new XMLHttpRequest();
	let {ContestID, ProblemCode} = match.params
	request.open("GET", "/api/v0/contests/" + ContestID + "/problems/" + ProblemCode, false);
	request.setRequestHeader("Content-Type", "application/json; charset=UTF-8");
	request.send();
	if (request.status !== 200) {
		return <NotFoundPage/>;
	}
	let problem: ContestProblem = JSON.parse(request.response);
	return (
		<Page title={problem.Title}>
			<div className="ui-block-wrap">
				<div className="ui-block">
					<div className="ui-block-header">
						<h2 className="title">{problem.Title}</h2>
					</div>
					<div className="ui-block-content">{problem.Description}</div>
				</div>
			</div>
		</Page>
	);
};

export default ContestProblemPage;
