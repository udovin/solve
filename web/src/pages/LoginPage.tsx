import React, {ReactNode} from "react";
import Page from "../layout/Page";
import Input from "../layout/Input";
import Button from "../layout/Button";

export default class LoginPage extends React.Component {
	render(): ReactNode {
		return (
			<Page title="Login">
				<div className="ui-block-wrap">
					<form className="ui-block" onSubmit={LoginPage.handleSubmit}>
						<div className="ui-block-header">
							<h2 className="title">Login</h2>
						</div>
						<div className="ui-block-content">
							<div className="ui-field">
								<label>
									<span className="label">Username:</span>
									<Input
										type="text" name="login"
										placeholder="Username" required
										autoFocus
									/>
								</label>
							</div>
							<div className="ui-field">
								<label>
									<span className="label">Password:</span>
									<Input
										type="password" name="password"
										placeholder="Password" required
									/>
								</label>
							</div>
						</div>
						<div className="ui-block-footer">
							<Button type="submit" color="primary">
								Login
							</Button>
						</div>
					</form>
				</div>
			</Page>
		);
	}

	static handleSubmit(event: any) {
		const login = event.target.login.value;
		const password = event.target.password.value;
		let request = new XMLHttpRequest();
		request.open("POST", "/api/v0/sessions");
		request.setRequestHeader("Content-Type", "application/json; charset=UTF-8");
		request.send(JSON.stringify({
			"Login": login,
			"Password": password,
		}));
		event.preventDefault();
	}
}
