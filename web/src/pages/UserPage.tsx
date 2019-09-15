import React, {useEffect, useState} from "react";
import {RouteComponentProps} from "react-router";
import Page from "../layout/Page";
import {User} from "../api";
import {Block} from "../layout/blocks";
import "./ContestPage.scss"
import Sidebar from "../layout/Sidebar";

type UserPageParams = {
	UserID: string;
}

const UserPage = ({match}: RouteComponentProps<UserPageParams>) => {
	const {UserID} = match.params;
	const [user, setUser] = useState<User>();
	useEffect(() => {
		fetch("/api/v0/users/" + UserID)
			.then(result => result.json())
			.then(result => setUser(result));
	}, [UserID]);
	if (!user) {
		return <>Loading...</>;
	}
	const {Login, FirstName, LastName, MiddleName} = user;
	return <Page title={Login} sidebar={<Sidebar/>}>
		<Block title={Login} id="block-user">
			{FirstName && <div className="ui-field">
				<label>
					<span className="label">First name:</span>
					<span>{FirstName}</span>
				</label>
			</div>}
			{LastName && <div className="ui-field">
				<label>
					<span className="label">Last name:</span>
					<span>{LastName}</span>
				</label>
			</div>}
			{MiddleName && <div className="ui-field">
				<label>
					<span className="label">Middle name:</span>
					<span>{MiddleName}</span>
				</label>
			</div>}
		</Block>
	</Page>;
};

export default UserPage;
