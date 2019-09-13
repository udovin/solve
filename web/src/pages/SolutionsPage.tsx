import React, {useEffect, useState} from "react";
import Page from "../layout/Page";
import {Solution} from "../api";
import "./ContestPage.scss"
import {SolutionsBlock} from "../layout/solutions";

const SolutionsPage = () => {
	const [solutions, setSolutions] = useState<Solution[]>();
	useEffect(() => {
		fetch("/api/v0/solutions")
			.then(result => result.json())
			.then(result => setSolutions(result));
	}, []);
	if (!solutions) {
		return <>Loading...</>;
	}
	return <Page title="Solutions">
		<SolutionsBlock title="Solutions" solutions={solutions}/>
	</Page>;
};

export default SolutionsPage;
