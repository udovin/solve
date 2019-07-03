import React, {ReactNode} from "react";
import {Page} from "../layout/Page";
import {Block} from "../layout/Block";

export class IndexPage extends React.Component {
	render(): ReactNode {
		return (
			<Page title="Index">
				<Block title="Index">

				</Block>
			</Page>
		);
	}
}
