import React, {useEffect, useState} from "react";
import Page from "../layout/Page";
import {RouteComponentProps} from "react-router";
import {Problem} from "../api";
import {Block} from "../layout/blocks";

type ProblemPageParams = {
	ProblemID: string;
}

const ProblemPage = ({match}: RouteComponentProps<ProblemPageParams>) => {
	const {ProblemID} = match.params;
	const [problem, setProblem] = useState<Problem>();
	useEffect(() => {
		fetch("/api/v0/problems/" + ProblemID)
			.then(result => result.json())
			.then(result => setProblem(result))
	}, [ProblemID]);
	if (problem) {
		return <Page title={problem.Title}>
			<Block title={problem.Title}>
				{problem.Description}
			</Block>
		</Page>;
	} else {
		return <>Loading...</>;
	}
};

export default ProblemPage;
