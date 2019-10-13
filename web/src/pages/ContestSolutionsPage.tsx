import React, {useEffect, useState} from "react";
import {RouteComponentProps} from "react-router";
import Page from "../components/Page";
import {Solution} from "../api";
import "./ContestPage.scss"
import {SolutionsBlock} from "../components/solutions";
import ContestTabs from "../components/ContestTabs";

type ContestPageParams = {
	ContestID: string;
}

const ContestSolutionsPage = ({match}: RouteComponentProps<ContestPageParams>) => {
	const {ContestID} = match.params;
	const [solutions, setSolutions] = useState<Solution[]>();
	useEffect(() => {
		fetch("/api/v0/contests/" + ContestID + "/solutions")
			.then(result => result.json())
			.then(result => setSolutions(result || []));
	}, [ContestID]);
	if (!solutions) {
		return <>Loading...</>;
	}
	return <Page title="Solutions">
		<ContestTabs contestID={+ContestID} pageType="solutions"/>
		<SolutionsBlock title="Solutions" solutions={solutions}/>
	</Page>;
};

export default ContestSolutionsPage;
