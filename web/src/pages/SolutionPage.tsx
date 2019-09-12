import React, {useContext, useEffect, useState} from "react";
import {RouteComponentProps} from "react-router";
import Page from "../layout/Page";
import {getDefense, getShortVerdict, Solution} from "../api";
import {Block} from "../layout/blocks";
import "./ContestPage.scss"
import {AuthContext} from "../AuthContext";
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
	if (!solution) {
		return <>Loading...</>;
	}
	let isSuper = Boolean(session && session.User.IsSuper);
	const {ID, Report} = solution;
	return <Page title={"Solution #" + solution.ID}>
		<Block title={"Solution #" + solution.ID} footer={<>
			{Report && <>
				<h3>Compile logs:</h3><pre><code>{Report.Data.CompileLogs.Stdout}</code></pre>
			</>}
			<h3>Source code:</h3>
			<pre><code>{solution.SourceCode}</code></pre>
		</>}>
			<table className="ui-table">
				<thead>
				<tr>
					<th>#</th>
					<th>Verdict</th>
					<th>Defense</th>
					<th>Points</th>
				</tr>
				</thead>
				<tbody>
				<tr>
					<td>{ID}</td>
					<td>{Report && getShortVerdict(Report.Verdict)}</td>
					<td>{Report && getDefense(Report.Data.Defense)}</td>
					<td>{Report && Report.Data.Points}</td>
				</tr>
				{isSuper && <tr>
					<td colSpan={2}>Изменить:</td>
					<td>
						<form onSubmit={updateVerdict}>
							<select name="verdict">
								<option value="1">{getDefense(1)}</option>
								<option value="2">{getDefense(2)}</option>
								<option value="3">{getDefense(3)}</option>
							</select>
							<Button type="submit">@</Button>
						</form>
					</td>
					<td>
						<form onSubmit={updatePoints}>
							<Input type="number" name="points"/>
							<Button type="submit">@</Button>
						</form>
					</td>
				</tr>}
				</tbody>
			</table>
		</Block>
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
