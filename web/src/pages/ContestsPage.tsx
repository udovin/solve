import React, {useEffect, useState} from "react";
import Page from "../components/Page";
import ContestsBlock from "../components/ContestsBlock";
import {Contest} from "../api";

const ContestsPage = () => {
	const [contests, setContests] = useState<Contest[]>([]);
	useEffect(() => {
		fetch("/api/v0/contests")
			.then(result => result.json())
			.then(result => setContests(result));
	}, []);
	return <Page title="Contests">
		{contests ?
			<ContestsBlock contests={contests}/> :
			<>Loading...</>}
	</Page>;
};

export default ContestsPage;
