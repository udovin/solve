import React, {ReactNode} from "react";
import {Link, RouteComponentProps, withRouter} from "react-router-dom";
import {AuthConsumer} from "../AuthContext";

class Header extends React.Component<RouteComponentProps> {
	render(): ReactNode {
		return (
			<header id="header">
				<div id="header-top">
					<div id="header-logo">
						<Link to="/">Solve</Link>
						<span>Online Judge</span>
					</div>
					<div id="header-account">
						<ul>
							<AuthConsumer>{this.getAccountLinks}</AuthConsumer>
						</ul>
					</div>
				</div>
				<nav id="header-nav">
					<ul>
						<li className={this.getActiveClass("/")}>
							<Link to="/">Index</Link>
						</li>
						<li className={this.getActiveClass("/contests")}>
							<Link to="/contests">Contests</Link>
						</li>
					</ul>
				</nav>
				<div id="header-version" title="Version">0.0.1</div>
			</header>
		);
	}

	public getAccountLinks(isAuth: boolean): ReactNode {
		if (isAuth) {
			return <>
				<li>
					<Link to="/profile">Profile</Link>
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

	public getActiveClass(...names: string[]): string {
		const {pathname} = this.props.location;
		for (let name of names) {
			if (name === pathname) {
				return "active";
			}
		}
		return "";
	}
}

export default withRouter(Header);
