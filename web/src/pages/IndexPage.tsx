import React from "react";
import Page from "../layout/Page";
import {Block} from "../layout/blocks";
import Sidebar from "../layout/Sidebar";

const IndexPage = () => {
	return <Page title="Index" sidebar={<Sidebar/>}>
		<Block title="Index">
		</Block>
	</Page>;
};

export default IndexPage;
