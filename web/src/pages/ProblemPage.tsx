import React, {ReactNode} from "react";
import Page from "../layout/Page";

export default class ProblemPage extends React.Component {
	render(): ReactNode {
		return (
			<Page title="Problem">
				<div className="ui-block-wrap">
					<div className="ui-block">
						<div className="ui-block-header">
							<h2 className="title">Problem</h2>
						</div>
						<div className="ui-block-content">

						</div>
					</div>
				</div>
			</Page>
		);
	}
}
