import React, {useEffect, useState} from "react";
import {Link} from "react-router-dom";
import Page from "../components/Page";
import Block from "../components/Block";

type Contest = {
	ID: number;
	UserID: number;
	Title: string;
	CreateTime: number;
}

const ContestsPage = () => {
	const [contests, setContests] = useState<Contest[]>([]);
	useEffect(() => {
		fetch("/api/v0/contests")
			.then(result => result.json())
			.then(result => setContests(result));
	}, []);
	return <Page title="Contests">
		<Block title="Contests">
			<ul>{contests && contests.map(
				(contest, index) => <li className="contest" key={index}>
					<Link to={"/contests/" + contest.ID}>
						{contest.Title}
					</Link>
				</li>
			)}</ul>
		</Block>
	</Page>;
};

export default ContestsPage;
