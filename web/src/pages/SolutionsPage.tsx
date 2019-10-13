import React, {useEffect, useState} from "react";
import Page from "../layout/Page";
import {Solution} from "../api";
import "./ContestPage.scss"
import {SolutionsBlock} from "../layout/solutions";
import NotFoundPage from "./NotFoundPage";

const SolutionsPage = () => {
	const [solutions, setSolutions] = useState<Solution[]>();
	const [notFound, setNotFound] = useState<boolean>(false);
	useEffect(() => {
		fetch("/api/v0/solutions")
			.then(result => result.json())
			.then(result => setSolutions(result))
			.catch(error => setNotFound(true));
	}, []);
	if (notFound) {
		return <NotFoundPage/>;
	}
	if (!solutions) {
		return <>Loading...</>;
	}
	return <Page title="Solutions">
		<SolutionsBlock title="Solutions" solutions={solutions}/>
	</Page>;
};

export default SolutionsPage;
