import React, {FC, useContext} from "react";
import {Link} from "react-router-dom";
import Block from "../ui/Block";
import {AuthContext} from "../AuthContext";

const Sidebar: FC = () => {
	const {status} = useContext(AuthContext);
	if (status && status.user) {
		const {login} = status.user;
		return <Block><ul>
			<li><Link to={`/users/${login}`}>Profile</Link></li>
			<li><Link to={`/users/${login}/edit`}>Edit</Link></li>
			<li><Link to="/logout">Logout</Link></li>
		</ul></Block>;
	}
	return <Block><ul>
		<li><Link to="/login">Login</Link></li>
		<li><Link to="/register">Register</Link></li>
	</ul></Block>;
};

export default Sidebar;
