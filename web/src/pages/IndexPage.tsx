import React, {ReactNode} from "react";
import Page from "../layout/Page";
import Block from "../layout/Block";
import Sidebar from "../layout/Sidebar";

export default class IndexPage extends React.Component {
	render(): ReactNode {
		return (
			<Page title="Index" sidebar={<Sidebar/>}>
				<Block title="Index">

				</Block>
			</Page>
		);
	}
}
