import React, {ReactNode} from "react";
import Page from "../layout/Page";
import Input from "../layout/Input";
import Button from "../layout/Button";

export default class RegisterPage extends React.Component {
	render(): ReactNode {
		return (
			<Page title="Register">
				<div className="ui-block-wrap">
					<form className="ui-block" onSubmit={RegisterPage.handleSubmit}>
						<div className="ui-block-header">
							<h2 className="title">Register</h2>
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
								<span className="text">
									You can use only English letters, digits, symbols &laquo;<code>_</code>&raquo; and &laquo;<code>-</code>&raquo;.
									Username can starts only with English letter and ends with English letter and digit.
								</span>
							</div>
							<div className="ui-field">
								<label>
									<span className="label">E-mail:</span>
									<Input
										type="text" name="email"
										placeholder="E-mail" required
									/>
								</label>
								<span className="text">
									You will receive an email to verify your account.
								</span>
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
							<div className="ui-field">
								<label>
									<span className="label">
										Repeat password:
									</span>
									<Input
										type="password" name="passwordRepeat"
										placeholder="Repeat password" required
									/>
								</label>
							</div>
						</div>
						<div className="ui-block-footer">
							<Button type="submit" color="primary">
								Register
							</Button>
						</div>
					</form>
				</div>
			</Page>
		);
	}

	static handleSubmit(event: any) {
		const login = event.target.login.value;
		const email = event.target.email.value;
		const password = event.target.password.value;
		let request = new XMLHttpRequest();
		request.open("POST", "/api/v0/users");
		request.setRequestHeader("Content-Type", "application/json; charset=UTF-8");
		request.send(JSON.stringify({
			"Login": login,
			"Email": email,
			"Password": password,
		}));
		event.preventDefault();
	}
}
