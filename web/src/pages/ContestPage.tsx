import React, {useEffect, useState} from "react";
import {RouteComponentProps} from "react-router";
import {Link} from "react-router-dom";
import Page from "../layout/Page";
import {Contest} from "../api";
import {Block} from "../layout/blocks";

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
	if (contest) {
		return (
			<Page title={contest.Title}>
				<Block title={contest.Title}>
					<ul>{contest.Problems.map(
						(problem) => <li className="problem">
							<Link
								to={"/contests/" + contest.ID + "/problems/" + problem.Code}>
								<span className="code">{problem.Code}</span>
								<span className="title">{problem.Title}</span>
							</Link>
						</li>
					)}</ul>
				</Block>
			</Page>
		);
	} else {
		return <>Loading...</>;
	}
};

export default ContestPage;
