import React, {useEffect, useState} from "react";
import {RouteComponentProps} from "react-router";
import {Link} from "react-router-dom";
import Page from "../layout/Page";
import {Contest} from "../api";
import {Block} from "../layout/blocks";
import "./ContestPage.scss"

type ContestPageParams = {
	ContestID: string;
}

const ContestPage = ({match}: RouteComponentProps<ContestPageParams>) => {
	const {ContestID} = match.params;
	const [contest, setContest] = useState<Contest>();
	useEffect(() => {
		fetch("/api/v0/contests/" + ContestID)
			.then(result => result.json())
			.then(result => setContest(result))
	}, [ContestID]);
	if (!contest) {
		return <>Loading...</>;
	}
	return <Page title={contest.Title}>
		<Block title={contest.Title} id="block-contest-problems">
			<table className="ui-table">
				<thead>
				<tr>
					<th className="id">#</th>
					<th className="name">Name</th>
				</tr>
				</thead>
				<tbody>{contest.Problems.map(
				(problem, index) => <tr className="problem" key={index}>
					<td className="id">
						<Link to={"/contests/" + contest.ID + "/problems/" + problem.Code}>{problem.Code}</Link>
					</td>
					<td className="name">
						<Link to={"/contests/" + contest.ID + "/problems/" + problem.Code}>{problem.Title}</Link>
					</td>
				</tr>
				)}</tbody>
			</table>
		</Block>
	</Page>;
};

export default ContestPage;
