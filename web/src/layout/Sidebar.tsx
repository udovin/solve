import React, {ReactNode} from "react";
import {Link} from "react-router-dom";
import {Block} from "./blocks";
import {AuthConsumer, AuthState} from "../AuthContext";

export default class Sidebar extends React.Component {
	render(): ReactNode {
		return <Block>
			<ul>
				<AuthConsumer>{Sidebar.getItems}</AuthConsumer>
			</ul>
		</Block>;
	}

	public static getItems(state: AuthState): ReactNode {
		if (state.session) {
			let login = state.session.User.Login;
			return <>
				<li>
					<Link to={"/users/"+login}>Profile</Link>
				</li>
				<li>
					<Link to="/logout">Logout</Link>
				</li>
			</>;
		}
		return <>
			<li>
				<Link to="/login">Login</Link>
			</li>
			<li>
				<Link to="/register">Register</Link>
			</li>
		</>;
	}
}
