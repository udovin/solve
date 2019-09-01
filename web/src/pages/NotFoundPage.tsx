import React from "react";
import Page from "../layout/Page";

const NotFoundPage = () => {
	return (
		<Page title="Page not found">
			<div className="ui-block-wrap">
				<div className="ui-block">
					<div className="ui-block-header">
						<h2 className="title">Page not found</h2>
					</div>
					<div className="ui-block-content">
					</div>
				</div>
			</div>
		</Page>
	);
};

export default NotFoundPage;
