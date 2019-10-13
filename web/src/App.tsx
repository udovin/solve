import React from "react";
import {BrowserRouter, Switch, Route} from "react-router-dom";
import "./App.scss";
import Header from "./components/Header";
import Footer from "./components/Footer";
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
import SolutionPage from "./pages/SolutionPage";
import UpdateProblemPage from "./pages/UpdateProblemPage";
import SolutionsPage from "./pages/SolutionsPage";
import UserPage from "./pages/UserPage";
import EditUserPage from "./pages/EditUserPage";
import ContestSolutionsPage from "./pages/ContestSolutionsPage";

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
						<Route exact path="/problems/:ProblemID/update" component={UpdateProblemPage}/>
						<Route exact path="/contests" component={ContestsPage}/>
						<Route exact path="/contests/create" component={CreateContestPage}/>
						<Route exact path="/contests/:ContestID" component={ContestPage}/>
						<Route exact path="/contests/:ContestID/solutions" component={ContestSolutionsPage}/>
						<Route exact path="/contests/:ContestID/problems/create" component={CreateContestProblemPage}/>
						<Route exact path="/contests/:ContestID/problems/:ProblemCode" component={ContestProblemPage}/>
						<Route exact path="/compilers/create" component={CreateCompilerPage}/>
						<Route exact path="/solutions" component={SolutionsPage}/>
						<Route exact path="/solutions/:SolutionID" component={SolutionPage}/>
						<Route exact path="/login" component={LoginPage}/>
						<Route exact path="/logout" component={LogoutPage}/>
						<Route exact path="/register" component={RegisterPage}/>
						<Route exact path="/language" component={LanguagePage}/>
						<Route exact path="/users/:UserID" component={UserPage}/>
						<Route exact path="/users/:UserID/edit" component={EditUserPage}/>
					</Switch>
					<Footer/>
				</BrowserRouter>
			</AuthProvider>
		</div>
	);
};

export default App;
