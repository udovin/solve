import React, {useEffect, useState} from "react";
import {RouteComponentProps} from "react-router";
import Page from "../components/Page";
import {ErrorResp, observeUser, User} from "../api";
import Block from "../ui/Block";
import "./ContestPage.scss"
import Sidebar from "../components/Sidebar";
import Field from "../ui/Field";
import Alert from "../ui/Alert";

type UserPageParams = {
	user_id: string;
}

const UserPage = ({match}: RouteComponentProps<UserPageParams>) => {
	const {user_id} = match.params;
	const [user, setUser] = useState<User>();
	const [error, setError] = useState<ErrorResp>({message: ""});
	useEffect(() => {
		observeUser(user_id)
			.then(user => {
				setError({message: ""});
				setUser(user);
			})
			.catch(error => setError(error));
	}, [user_id]);
	if (error.message) {
		return <Alert>{error.message}</Alert>;
	}
	if (!user) {
		return <>Loading...</>;
	}
	const {login, email, first_name, last_name, middle_name} = user;
	return <Page title={login} sidebar={<Sidebar/>}>
		<Block title={login} id="block-user">
			{email && <Field title="E-mail:"><span>{email}</span></Field>}
			{first_name && <Field title="First name:"><span>{first_name}</span></Field>}
			{last_name && <Field title="Last name:"><span>{last_name}</span></Field>}
			{middle_name && <Field title="Middle name:"><span>{middle_name}</span></Field>}
		</Block>
	</Page>;
};

export default UserPage;
