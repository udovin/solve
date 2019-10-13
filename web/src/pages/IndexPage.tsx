import React from "react";
import Page from "../components/Page";
import Block from "../components/Block";
import Sidebar from "../components/Sidebar";

const IndexPage = () => {
	return <Page title="Index" sidebar={<Sidebar/>}>
		<Block title="Index">
		</Block>
	</Page>;
};

export default IndexPage;
