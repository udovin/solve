import React from "react";
import {BrowserRouter, Switch, Route} from "react-router-dom";
import "./App.scss";
import Header from "./layout/Header";
import Footer from "./layout/Footer";
import LoginPage from "./pages/LoginPage";
import RegisterPage from "./pages/RegisterPage";
import IndexPage from "./pages/IndexPage";
import LanguagePage from "./pages/LanguagePage";
import {AuthProvider} from "./AuthContext";
import CreateProblemPage from "./pages/CreateProblemPage";
import ProblemPage from "./pages/ProblemPage";
import ContestsPage from "./pages/ContestsPage";
import CreateContestPage from "./pages/CreateContestPage";
import ContestPage from "./pages/ContestPage";
import ContestProblemPage from "./pages/ContestProblemPage";
import CreateCompilerPage from "./pages/CreateCompilerPage";
import LogoutPage from "./pages/LogoutPage";
import CreateContestProblemPage from "./pages/CreateContestProblemPage";

const App: React.FC = () => {
	return (
		<div id="layout">
			<AuthProvider>
				<BrowserRouter>
					<Header/>
					<Switch>
						<Route exact path="/" component={IndexPage}/>
						<Route exact path="/problems/create" component={CreateProblemPage}/>
						<Route exact path="/problems/:ProblemID" component={ProblemPage}/>
						<Route exact path="/contests" component={ContestsPage}/>
						<Route exact path="/contests/create" component={CreateContestPage}/>
						<Route exact path="/contests/:ContestID" component={ContestPage}/>
						<Route exact path="/contests/:ContestID/problems/create" component={CreateContestProblemPage}/>
						<Route exact path="/contests/:ContestID/problems/:ProblemCode" component={ContestProblemPage}/>
						<Route exact path="/compilers/create" component={CreateCompilerPage}/>
						<Route exact path="/login" component={LoginPage}/>
						<Route exact path="/logout" component={LogoutPage}/>
						<Route exact path="/register" component={RegisterPage}/>
						<Route exact path="/language" component={LanguagePage}/>
					</Switch>
					<Footer/>
				</BrowserRouter>
			</AuthProvider>
		</div>
	);
};

export default App;
