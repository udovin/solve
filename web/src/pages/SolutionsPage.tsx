import React, {useEffect, useState} from "react";
import Page from "../components/Page";
import {Solution} from "../api";
import "./ContestPage.scss"
import {SolutionsBlock} from "../components/solutions";
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
	return <Page title="Solutions">
		{solutions ?
			<SolutionsBlock title="Solutions" solutions={solutions}/> :
			<>Loading...</>}
	</Page>;
};

export default SolutionsPage;
