import React, {useEffect, useState} from "react";
import {RouteComponentProps} from "react-router";
import Page from "../layout/Page";
import {Solution} from "../api";
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
	return <Page title={"Solution #" + solution.ID}>
		<Block title={"Solution #" + solution.ID}>
			<pre><code>{solution.SourceCode}</code></pre>
		</Block>
	</Page>;
};

export default SolutionPage;
