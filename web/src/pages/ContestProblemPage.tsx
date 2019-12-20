import React, {useEffect, useState} from "react";
import {Redirect, RouteComponentProps} from "react-router";
import Page from "../components/Page";
import {Compiler, ContestProblem, Solution} from "../api";
import Block from "../components/Block";
import {SolutionsSideBlock, SubmitSolutionSideBlock} from "../components/solutions";

type ContestProblemPageParams = {
	ContestID: string;
	ProblemCode: string;
}

const ContestProblemPage = ({match}: RouteComponentProps<ContestProblemPageParams>) => {
	const {ContestID, ProblemCode} = match.params;
	const [problem, setProblem] = useState<ContestProblem>();
	const [compilers, setCompilers] = useState<Compiler[]>();
	const [solution, setSolution] = useState<Solution>();
	const onSubmit = (event: any) => {
		event.preventDefault();
		const {sourceFile, sourceText, compilerID} = event.target;
		let create = (code: string) => {
			fetch("/api/v0/contests/" + ContestID + "/problems/" + ProblemCode, {
				method: "POST",
				headers: {
					"Content-Type": "application/json; charset=UTF-8",
				},
				body: JSON.stringify({
					CompilerID: Number(compilerID.value),
					SourceCode: code,
				})
			})
				.then(result => result.json())
				.then(result => setSolution(result));
		};
		if (sourceFile.files.length > 0) {
			let reader = new FileReader();
			reader.onload = (event: any) => create(event.target.result);
			reader.readAsText(sourceFile.files[0]);
		} else {
			create(sourceText.value);
		}
	};
	useEffect(() => {
		fetch("/api/v0/compilers")
			.then(result => result.json())
			.then(result => setCompilers(result))
	}, []);
	useEffect(() => {
		fetch("/api/v0/contests/" + ContestID + "/problems/" + ProblemCode)
			.then(result => result.json())
			.then(result => setProblem(result));
	}, [ContestID, ProblemCode]);
	if (solution) {
		return <Redirect to={"/solutions/" + solution.ID} push={true}/>;
	}
	if (!problem) {
		return <>Loading...</>;
	}
	return <Page title={problem.Title} sidebar={
		<>
			<SubmitSolutionSideBlock onSubmit={onSubmit} compilers={compilers}/>
			{problem.Solutions && <SolutionsSideBlock solutions={problem.Solutions}/>}
		</>
	}>
		<Block title={problem.Title}>
			<div className="problem-statement" dangerouslySetInnerHTML={{__html: problem.Description}}/>
		</Block>
	</Page>;
};

export default ContestProblemPage;
