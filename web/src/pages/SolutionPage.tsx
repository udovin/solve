import React, {useEffect, useState} from "react";
import {RouteComponentProps} from "react-router";
import Page from "../layout/Page";
import {getShortVerdict, Solution} from "../api";
import {Block} from "../layout/blocks";
import "./ContestPage.scss"

type SolutionPageParams = {
	SolutionID: string;
}

const SolutionPage = ({match}: RouteComponentProps<SolutionPageParams>) => {
	const {SolutionID} = match.params;
	const [solution, setSolution] = useState<Solution>();
	useEffect(() => {
		fetch("/api/v0/solutions/" + SolutionID)
			.then(result => result.json())
			.then(result => setSolution(result));
	}, [SolutionID]);
	if (!solution) {
		return <>Loading...</>;
	}
	const {ID, Report} = solution;
	return <Page title={"Solution #" + solution.ID}>
		<Block title={"Solution #" + solution.ID} footer={
			<pre><code>{solution.SourceCode}</code></pre>
		}>
			<table className="ui-table">
				<thead>
				<tr>
					<th>#</th>
					<th>Verdict</th>
				</tr>
				</thead>
				<tbody>
				<tr>
					<td>{ID}</td>
					<td>{Report && getShortVerdict(Report.Verdict)}</td>
				</tr>
				</tbody>
			</table>
		</Block>
	</Page>;
};

export default SolutionPage;
