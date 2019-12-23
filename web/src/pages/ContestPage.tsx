import React, {FC, useEffect, useState} from "react";
import {Redirect, Route, RouteComponentProps, Switch} from "react-router";
import {Link} from "react-router-dom";
import Page from "../components/Page";
import {Compiler, Contest, ContestProblem, Solution} from "../api";
import Block from "../components/Block";
import ContestTabs from "../components/ContestTabs";
import "./ContestPage.scss"
import {
	SolutionsBlock,
	SolutionsSideBlock,
	SubmitSolutionSideBlock
} from "../components/solutions";
import FormBlock from "../components/FormBlock";
import Button from "../components/Button";
import Field from "../components/Field";
import Input from "../components/Input";

type ContestPageParams = {
	ContestID: string;
}

type ContestBlockParams = {
	contest: Contest;
};

const ContestProblemsBlock: FC<ContestBlockParams> = props => {
	const {contest} = props;
	const {ID, Title, Problems} = contest;
	return <Block title={Title} id="block-contest-problems">
		<table className="ui-table">
			<thead>
			<tr>
				<th className="id">#</th>
				<th className="name">Name</th>
			</tr>
			</thead>
			<tbody>{Problems && Problems.map(
				(problem, index) => <tr className="problem" key={index}>
					<td className="id">
						<Link to={"/contests/" + ID + "/problems/" + problem.Code}>{problem.Code}</Link>
					</td>
					<td className="name">
						<Link to={"/contests/" + ID + "/problems/" + problem.Code}>{problem.Title}</Link>
					</td>
				</tr>
			)}</tbody>
		</table>
	</Block>;
};

const ContestSolutionsBlock: FC<ContestBlockParams> = props => {
	const {contest} = props;
	const [solutions, setSolutions] = useState<Solution[]>();
	useEffect(() => {
		fetch("/api/v0/contests/" + contest.ID + "/solutions")
			.then(result => result.json())
			.then(result => setSolutions(result || []));
	}, [contest.ID]);
	if (!solutions) {
		return <>Loading...</>;
	}
	return <SolutionsBlock title="Solutions" solutions={solutions}/>;
};

const CreateContestProblemBlock = ({match}: RouteComponentProps<ContestPageParams>) => {
	const {ContestID} = match.params;
	const [success, setSuccess] = useState<boolean>();
	const onSubmit = (event: any) => {
		event.preventDefault();
		const {problemID, code} = event.target;
		fetch("/api/v0/contests/" + ContestID + "/problems", {
			method: "POST",
			headers: {
				"Content-Type": "application/json; charset=UTF-8",
			},
			body: JSON.stringify({
				ProblemID: Number(problemID.value),
				Code: code.value,
			})
		})
			.then(() => setSuccess(true));
	};
	if (success) {
		return <Redirect to={"/contests/" + ContestID}/>
	}
	return <FormBlock onSubmit={onSubmit} title="Add contest problem" footer={
		<Button type="submit" color="primary">Create</Button>
	}>
		<Field title="Problem ID:">
			<Input type="number" name="problemID" placeholder="ID" required autoFocus/>
		</Field>
		<Field title="Code:">
			<Input type="text" name="code" placeholder="Code" required/>
		</Field>
	</FormBlock>;
};

type ContestProblemPageParams = ContestPageParams & {
	ProblemCode: string;
}

const ContestProblemSideBlock = ({match}: RouteComponentProps<ContestProblemPageParams>) => {
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
	return <>
		<SubmitSolutionSideBlock onSubmit={onSubmit} compilers={compilers}/>
		{problem.Solutions && <SolutionsSideBlock solutions={problem.Solutions}/>}
	</>;
};

const ContestProblemBlock = ({match}: RouteComponentProps<ContestProblemPageParams>) => {
	const {ContestID, ProblemCode} = match.params;
	const [problem, setProblem] = useState<ContestProblem>();
	useEffect(() => {
		fetch("/api/v0/contests/" + ContestID + "/problems/" + ProblemCode)
			.then(result => result.json())
			.then(result => setProblem(result));
	}, [ContestID, ProblemCode]);
	if (!problem) {
		return <>Loading...</>;
	}
	return <Block title={problem.Title}>
		<div className="problem-statement" dangerouslySetInnerHTML={{__html: problem.Description}}/>
	</Block>;
};

const ContestPage = ({match}: RouteComponentProps<ContestPageParams>) => {
	const {ContestID} = match.params;
	const [contest, setContest] = useState<Contest>();
	const [currentTab, setCurrentTab] = useState<string>();
	useEffect(() => {
		fetch("/api/v0/contests/" + ContestID)
			.then(result => result.json())
			.then(result => setContest(result));
	}, [ContestID]);
	if (!contest) {
		return <>Loading...</>;
	}
	const {Title} = contest;
	return <Page title={Title} sidebar={<Switch>
		<Route exact path="/contests/:ContestID/problems/:ProblemCode" component={ContestProblemSideBlock}/>
	</Switch>}>
		<ContestTabs contestID={contest.ID} currentTab={currentTab}/>
		<Switch>
			<Route exact path="/contests/:ContestID">
				{() => {
					setCurrentTab("problems");
					return <ContestProblemsBlock contest={contest} />;
				}}
			</Route>
			<Route exact path="/contests/:ContestID/solutions">
				{() => {
					setCurrentTab("solutions");
					return <ContestSolutionsBlock contest={contest} />;
				}}
			</Route>
			<Route exact path="/contests/:ContestID/problems/create" component={CreateContestProblemBlock}/>
			<Route exact path="/contests/:ContestID/problems/:ProblemCode" component={ContestProblemBlock}/>
		</Switch>
	</Page>;
};

export default ContestPage;
