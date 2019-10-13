import React, {FC, ReactNode} from "react";

export type PageProps = {
	title: string;
	sidebar?: ReactNode;
};

const Page: FC<PageProps> = props => {
	const {title, sidebar, children} = props;
	document.title = title;
	return <main id="main">
		{sidebar && <div id="sidebar-wrap"><div id="sidebar">{sidebar}</div></div>}
		<div id="content-wrap"><div id="content">{children}</div></div>
	</main>;
};

export default Page;
