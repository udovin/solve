import React, {useContext} from "react";
import Page from "../components/Page";
import Input from "../ui/Input";
import Button from "../ui/Button";
import FormBlock from "../components/FormBlock";
import {Redirect} from "react-router";
import {AuthContext} from "../AuthContext";
import Field from "../components/Field";
import {authStatus, loginUser} from "../api";

const LoginPage = () => {
	const {status, setStatus} = useContext(AuthContext);
	const onSubmit = (event: any) => {
		event.preventDefault();
		const {Login, Password} = event.target;
		loginUser({
			Login: Login.value,
			Password: Password.value,
		})
			.then(() => {
				authStatus()
					.then(result => result.json())
					.then(result => setStatus(result));
			});
	};
	if (status && status.User) {
		return <Redirect to={"/"}/>
	}
	return <Page title="Login">
		<FormBlock onSubmit={onSubmit} title="Login" footer={
			<Button type="submit" color="primary">Login</Button>
		}>
			<Field title="Username:">
				<Input type="text" name="Login" placeholder="Username" required autoFocus/>
			</Field>
			<Field title="Password:">
				<Input type="password" name="Password" placeholder="Password" required/>
			</Field>
		</FormBlock>
	</Page>;
};

export default LoginPage;
