import React, {useContext, useEffect, useState} from "react";
import {RouteComponentProps} from "react-router";
import {AuthContext} from "../AuthContext";
import {getDefense, getShortVerdict, Solution} from "../api";
import Page from "../components/Page";
import Block from "../components/Block";
import Input from "../ui/Input";
import Button from "../ui/Button";
import SolutionRow from "../components/SolutionRow";
import "./ContestPage.scss"

type SolutionPageParams = {
	SolutionID: string;
}

const SolutionPage = ({match}: RouteComponentProps<SolutionPageParams>) => {
	const {SolutionID} = match.params;
	const [solution, setSolution] = useState<Solution>();
	const {status} = useContext(AuthContext);
	useEffect(() => {
		fetch("/api/v0/solutions/" + SolutionID)
			.then(result => result.json())
			.then(result => setSolution(result));
	}, [SolutionID]);
	const updateVerdict = (event: any) => {
		event.preventDefault();
		const {verdict} = event.target;
		fetch("/api/v0/solutions/" + SolutionID + "/report", {
			method: "POST",
			headers: {
				"Content-Type": "application/json; charset=UTF-8",
			},
			body: JSON.stringify({
				Defense: Number(verdict.value),
			}),
		})
			.then(() => document.location.reload());
	};
	const updatePoints = (event: any) => {
		event.preventDefault();
		const {points} = event.target;
		fetch("/api/v0/solutions/" + SolutionID + "/report", {
			method: "POST",
			headers: {
				"Content-Type": "application/json; charset=UTF-8",
			},
			body: JSON.stringify({
				Points: Number(points.value),
			}),
		})
			.then(() => document.location.reload());
	};
	const rejudge = (event: any) => {
		event.preventDefault();
		fetch("/api/v0/solutions/" + SolutionID, {
			method: "POST",
		}).then(() => document.location.reload());
	};
	if (!solution) {
		return <>Loading...</>;
	}
	let isSuper = Boolean(status && status.User.IsSuper);
	const {Report} = solution;
	return <Page title={"Solution #" + solution.ID}>
		<Block title={"Solution #" + solution.ID} className="b-solutions">
			<table className="ui-table">
				<thead>
				<tr>
					<th className="created">Created</th>
					<th className="participant">Participant</th>
					<th className="problem">Problem</th>
					<th className="verdict">Verdict</th>
				</tr>
				</thead>
				<tbody>
				<SolutionRow showID={false} solution={solution}/>
				</tbody>
			</table>
		</Block>
		{isSuper && <Block title="Administration">
			<form onSubmit={rejudge}>
				<Button type="submit">Rejudge</Button>
			</form>
			<form onSubmit={updateVerdict}>
				<select name="verdict">
					<option value="1">{getDefense(1)}</option>
					<option value="2">{getDefense(2)}</option>
					<option value="3">{getDefense(3)}</option>
				</select>
				<Button type="submit">@</Button>
			</form>
			<form onSubmit={updatePoints}>
				<Input type="number" name="points"/>
				<Button type="submit">@</Button>
			</form>
		</Block>}
		<Block title="Source code">
			<pre><code>{solution.SourceCode}</code></pre>
		</Block>
		{Report && <Block title="Compilation">
			<pre><code>{Report.Data.CompileLogs.Stdout}</code></pre>
		</Block>}
		{Report && <Block title="Tests">
			<table className="ui-table">
				<thead>
				<tr>
					<th>Stderr</th>
					<th>Verdict</th>
				</tr>
				</thead>
				<tbody>{Report.Data.Tests && Report.Data.Tests.map((test, index) => <tr key={index}>
					<td>{test.CheckLogs.Stderr}</td>
					<td>{getShortVerdict(test.Verdict)}</td>
				</tr>)}</tbody>
			</table>
		</Block>}
	</Page>;
};

export default SolutionPage;
