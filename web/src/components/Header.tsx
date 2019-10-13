import React, {FC, useContext} from "react";
import {Link, RouteComponentProps, withRouter} from "react-router-dom";
import {AuthContext} from "../AuthContext";

const Header: FC<RouteComponentProps> = props => {
	const getActiveClass = (...names: string[]): string => {
		const {pathname} = props.location;
		for (let name of names) {
			if (name === pathname) {
				return "active";
			}
		}
		return "";
	};
	const {session} = useContext(AuthContext);
	let accountLinks = <>
		<li>
			<Link to="/login">Login</Link>
		</li>
		<li>
			<Link to="/register">Register</Link>
		</li>
	</>;
	if (session) {
		const {Login} = session.User;
		accountLinks = <>
			<li>
				Hello, <Link to={`/users/${Login}`}>{Login}</Link>!
			</li>
			<li>
				<Link to="/logout">Logout</Link>
			</li>
		</>;
	}
	return <header id="header">
		<div id="header-top">
			<div id="header-logo">
				<Link to="/">Solve</Link>
				<span>Online Judge</span>
			</div>
			<div id="header-account">
				<ul>
					{accountLinks}
				</ul>
			</div>
		</div>
		<nav id="header-nav">
			<ul>
				<li className={getActiveClass("/")}>
					<Link to="/">Index</Link>
				</li>
				<li className={getActiveClass("/contests")}>
					<Link to="/contests">Contests</Link>
				</li>
				<li className={getActiveClass("/solutions")}>
					<Link to="/solutions">Solutions</Link>
				</li>
			</ul>
		</nav>
		<div id="header-version" title="Version">0.0.1</div>
	</header>;
};

export default withRouter(Header);
