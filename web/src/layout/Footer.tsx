import React, {ReactNode} from "react";
import {Link} from "react-router-dom";

export default class Footer extends React.Component {
	render(): ReactNode {
		return (
			<footer id="footer">
				<div id="footer-nav">
					<div className="wrap">
						<ul>
							<li>
								<a href="//github.com/udovin/solve">Repository</a>
							</li>
							<li>Language: <Link to="/language">English</Link></li>
						</ul>
					</div>
				</div>
				<div id="footer-copy">
					<a href="//vk.com/wilcot">Ivan Udovin</a> &copy; 2019
				</div>
			</footer>
		);
	}
}
