import React from "react";
import Page from "../layout/Page";
import Input from "../layout/Input";
import {Button} from "../layout/buttons";
import {FormBlock} from "../layout/blocks";

const LoginPage = () => {
	let onSubmit = (event: any) => {
		event.preventDefault();
		const {login, password} = event.target;
		fetch("/api/v0/session", {
			method: "POST",
			headers: {
				"Content-Type": "application/json; charset=UTF-8",
			},
			body: JSON.stringify({
				Login: login.value,
				Password: password.value,
			})
		}).then();
	};
	return <Page title="Login">
		<FormBlock onSubmit={onSubmit} title="Login" footer={
			<Button type="submit" color="primary">Login</Button>
		}>
			<div className="ui-field">
				<label>
					<span className="label">Username:</span>
					<Input type="text" name="login" placeholder="Username" required autoFocus/>
				</label>
			</div>
			<div className="ui-field">
				<label>
					<span className="label">Password:</span>
					<Input type="password" name="password" placeholder="Password" required/>
				</label>
			</div>
		</FormBlock>
	</Page>;
};

export default LoginPage;
