import React, {FC, useContext} from "react";
import {Link} from "react-router-dom";
import {Block} from "./blocks";
import {AuthContext} from "../AuthContext";

const Sidebar: FC = () => {
	const {session} = useContext(AuthContext);
	let items = <>
		<li>
			<Link to="/login">Login</Link>
		</li>
		<li>
			<Link to="/register">Register</Link>
		</li>
	</>;
	if (session) {
		let login = session.User.Login;
		items = <>
			<li>
				<Link to={"/users/"+login}>Profile</Link>
			</li>
			<li>
				<Link to={"/users/"+login+"/edit"}>Edit</Link>
			</li>
			<li>
				<Link to="/logout">Logout</Link>
			</li>
		</>;
	}
	return <Block><ul>{items}</ul></Block>;
};

export default Sidebar;
