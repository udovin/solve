import React, {FC, useEffect, useState} from "react";
import {authStatus, AuthStatus} from "./api";

type Auth = {
	status?: AuthStatus;
	setStatus(status?: AuthStatus): void;
};

const AuthContext = React.createContext<Auth>({
	setStatus(): void {}
});

const AuthProvider: FC = props => {
	const [status, setStatus] = useState<AuthStatus>();
	useEffect(() => {
		authStatus()
			.then(result => result.json())
			.then(result => setStatus(result))
			.catch(error => setStatus(undefined))
	}, []);
	return <AuthContext.Provider value={{status, setStatus}}>
		{props.children}
	</AuthContext.Provider>;
};

export {AuthContext, AuthProvider};
