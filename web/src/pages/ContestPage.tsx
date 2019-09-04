import React from "react";
import {RouteComponentProps} from "react-router";
import {Link} from "react-router-dom";
import Page from "../layout/Page";
import NotFoundPage from "./NotFoundPage";
import {Contest} from "../api";

type ContestPageParams = {
	ContestID: string;
}

const ContestPage = ({match}: RouteComponentProps<ContestPageParams>) => {
	let request = new XMLHttpRequest();
	request.open("GET", "/api/v0/contests/" + match.params.ContestID, false);
	request.setRequestHeader("Content-Type", "application/json; charset=UTF-8");
	request.send();
	if (request.status !== 200) {
		return <NotFoundPage/>;
	}
	let contest: Contest = JSON.parse(request.response);
	return (
		<Page title={contest.Title}>
			<div className="ui-block-wrap">
				<div className="ui-block">
					<div className="ui-block-header">
						<h2 className="title">{contest.Title}</h2>
					</div>
					<div className="ui-block-content">
						<ul>{contest.Problems.map(
							(problem) => <li className="problem">
								<Link to={"/contests/" + contest.ID + "/problems/" + problem.Code}>
									<span className="code">{problem.Code}</span>
									<span className="title">{problem.Title}</span>
								</Link>
							</li>
						)}</ul>
					</div>
				</div>
			</div>
		</Page>
	);
};

export default ContestPage;
