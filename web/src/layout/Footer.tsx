import React from "react";
import {Link} from "react-router-dom";

export class Footer extends React.Component {
	render() {
		return (
			<footer id="footer">
				<div id="footer-nav">
					<div className="wrap">
						<ul>
							<li>
								<a href="//github.com/udovin/Solve-Web">Repository</a>
							</li>
							<li>Language: <Link to="/language">English</Link></li>
						</ul>
					</div>
				</div>
				<div id="footer-copy">
					<a href="//vk.com/wilcot">Ivan Udovin &copy; 2019</a>
				</div>
			</footer>
		);
	}
}
