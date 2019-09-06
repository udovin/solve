import React, {useEffect, useState} from "react";
import {RouteComponentProps} from "react-router";
import Page from "../layout/Page";
import {ContestProblem} from "../api";

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
	if (problem) {
		return <Page title={problem.Title}>
			<div className="ui-block-wrap">
				<div className="ui-block">
					<div className="ui-block-header">
						<h2 className="title">{problem.Title}</h2>
					</div>
					<div className="ui-block-content">
						{problem.Description}
					</div>
				</div>
			</div>
		</Page>;
	} else {
		return <>Loading...</>;
	}
};

export default ContestProblemPage;
