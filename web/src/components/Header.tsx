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
	const {status} = useContext(AuthContext);
	const accountLinks = <>
		{status && status.user && <li>
			Hello, <Link to={`/users/${status.user.login}`}>{status.user.login}</Link>!
		</li>}
		{(!status || (!status.session && status.roles.includes("login"))) && <li>
			<Link to="/login">Login</Link>
		</li>}
		{status && status.session && status.roles.includes("logout") && <li>
			<Link to="/logout">Logout</Link>
		</li>}
		{(!status || status.roles.includes("register")) && <li>
			<Link to="/register">Register</Link>
		</li>}
	</>;
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
