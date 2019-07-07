import React, {ReactNode} from "react";
import {Link} from "react-router-dom";
import Block from "./Block";

export default class Sidebar extends React.Component {
	render(): ReactNode {
		return <>
			<Block>
				<ul>
					<Link to="/profile">Profile</Link>
				</ul>
			</Block>
		</>;
	}
}
