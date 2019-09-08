import React, {useEffect, useState} from "react";
import {RouteComponentProps} from "react-router";
import Page from "../layout/Page";
import {ContestProblem} from "../api";
import {Block} from "../layout/blocks";

type ContestProblemPageParams = {
	ContestID: string;
	ProblemCode: string;
}

const ContestProblemPage = ({match}: RouteComponentProps<ContestProblemPageParams>) => {
	const {ContestID, ProblemCode} = match.params;
	const [problem, setProblem] = useState<ContestProblem>();
	useEffect(() => {
		fetch("/api/v0/contests/" + ContestID + "/problems/" + ProblemCode)
			.then(result => result.json())
			.then(result => setProblem(result))
	}, [ContestID, ProblemCode]);
	if (!problem) {
		return <>Loading...</>;
	}
	return <Page title={problem.Title}>
		<Block title={problem.Title}>
			<div className="problem-statement" dangerouslySetInnerHTML={{__html: problem.Description}}/>
		</Block>
	</Page>;
};

export default ContestProblemPage;
