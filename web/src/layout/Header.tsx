import React from "react";
import {Link} from "react-router-dom";

export class Header extends React.Component {
	render() {
		return (
			<header id="header">
				<div id="header-top">
					<div id="header-logo">
						<Link to="/">Solve</Link>
						<span>Online Judge</span>
					</div>
					<div id="header-account">
						<ul>
							<li>
								<Link to="/login">Login</Link>
							</li>
							<li>
								<Link to="/register">Register</Link>
							</li>
						</ul>
					</div>
				</div>
				<nav id="header-nav">
					<ul>
						<li className="active">
							<Link to="/">Index</Link>
						</li>
						<li className="">
							<Link to="/contests">Contests</Link>
						</li>
					</ul>
				</nav>
				<div id="header-version" title="Version">0.0.1</div>
			</header>
		);
	}
}
