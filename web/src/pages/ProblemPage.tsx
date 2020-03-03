import React, {useEffect, useState} from "react";
import Page from "../components/Page";
import {RouteComponentProps} from "react-router";
import {Problem} from "../api";
import Block from "../ui/Block";

type ProblemPageParams = {
	ProblemID: string;
}

const ProblemPage = ({match}: RouteComponentProps<ProblemPageParams>) => {
	const {ProblemID} = match.params;
	const [problem, setProblem] = useState<Problem>();
	useEffect(() => {
		fetch("/api/v0/problems/" + ProblemID)
			.then(result => result.json())
			.then(result => setProblem(result));
	}, [ProblemID]);
	if (!problem) {
		return <>Loading...</>;
	}
	return <Page title={problem.Title}>
		<Block title={problem.Title}>
			<div className="problem-statement" dangerouslySetInnerHTML={{__html: problem.Description}}/>
		</Block>
	</Page>;
};

export default ProblemPage;
