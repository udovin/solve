import React, {FC, useContext} from "react";
import {Link} from "react-router-dom";
import Block from "./Block";
import {AuthContext} from "../AuthContext";

const Sidebar: FC = () => {
	const {session} = useContext(AuthContext);
	if (session) {
		const {Login} = session.User;
		return <Block><ul>
			<li><Link to={`/users/${Login}`}>Profile</Link></li>
			<li><Link to={`/users/${Login}/edit`}>Edit</Link></li>
			<li><Link to="/logout">Logout</Link></li>
		</ul></Block>;
	}
	return <Block><ul>
		<li><Link to="/login">Login</Link></li>
		<li><Link to="/register">Register</Link></li>
	</ul></Block>;
};

export default Sidebar;
