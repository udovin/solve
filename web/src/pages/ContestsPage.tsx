import React from "react";
import {Link} from "react-router-dom";
import Page from "../layout/Page";
import {Block} from "../layout/blocks";
import NotFoundPage from "./NotFoundPage";

type Contest = {
	ID: number;
	UserID: number;
	Title: string;
	CreateTime: number;
}

const ContestsPage = () => {
	let request = new XMLHttpRequest();
	request.open("GET", "/api/v0/contests", false);
	request.setRequestHeader("Content-Type", "application/json; charset=UTF-8");
	request.send();
	if (request.status !== 200) {
		return <NotFoundPage/>;
	}
	let contests: Contest[] = JSON.parse(request.response);
	return (
		<Page title="Contests">
			<Block title="Contests">
				<ul>
					{contests.map((contest) => <li className="contest">
						<Link to={"/contests/" + contest.ID}>
							{contest.Title}
						</Link>
					</li>)}
				</ul>
			</Block>
		</Page>
	);
};

export default ContestsPage;
