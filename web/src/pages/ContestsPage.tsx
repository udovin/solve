import React, {useEffect, useState} from "react";
import {Link} from "react-router-dom";
import Page from "../layout/Page";
import {Block} from "../layout/blocks";

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
			.then(result => setContests(result))
	}, []);
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
