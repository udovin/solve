import React, {useContext, useEffect, useState} from "react";
import {RouteComponentProps} from "react-router";
import Page from "../layout/Page";
import {getDefense, getShortVerdict, Solution} from "../api";
import {Block} from "../layout/blocks";
import "./ContestPage.scss"
import {AuthContext} from "../AuthContext";
import {Link} from "react-router-dom";
import {Button} from "../layout/buttons";
import Input from "../layout/Input";

type SolutionPageParams = {
	SolutionID: string;
}

const SolutionPage = ({match}: RouteComponentProps<SolutionPageParams>) => {
	const {SolutionID} = match.params;
	const [solution, setSolution] = useState<Solution>();
	const {session} = useContext(AuthContext);
	useEffect(() => {
		fetch("/api/v0/solutions/" + SolutionID)
			.then(result => result.json())
			.then(result => setSolution(result));
	}, [SolutionID]);
	const format = (n: number) => {
		return ("0" + n).slice(-Math.max(2, String(n).length));
	};
	const formatDate = (d: Date) =>
		[d.getFullYear(), d.getMonth() + 1, d.getDate()].map(format).join("-");
	const formatTime = (d: Date) =>
		[d.getHours(), d.getMinutes(), d.getSeconds()].map(format).join(":");
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
	let isSuper = Boolean(session && session.User.IsSuper);
	const {CreateTime, User, Problem, Report} = solution;
	let createDate = new Date(CreateTime * 1000);
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
				<tr>
					<td className="created">
						<div className="time">{formatTime(createDate)}</div>
						<div className="date">{formatDate(createDate)}</div>
					</td>
					<td className="participant">{User ?
						<Link to={"/users/" + User.Login}>{User.Login}</Link> :
						<span>&mdash;</span>
					}</td>
					<td className="problem">{Problem ?
						<Link to={"/problems/" + Problem.ID}>{Problem.Title}</Link> :
						<span>&mdash;</span>
					}</td>
					<td className="verdict">
						<div className="type">{Report && getShortVerdict(Report.Verdict)}</div>
						<div className="value">{Report && Report.Data.Points}</div>
						<div className="defense">{Report && getDefense(Report.Data.Defense)}</div>
					</td>
				</tr>
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
