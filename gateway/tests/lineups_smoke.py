#!/usr/bin/env python3
"""Real gateway smoke test. Creates two unique users and deletes its saved lineups.

Run only after the integrated gateway, auth, stats, simulation and lineups services
are ready. BASE_URL defaults to http://localhost:8080. Accounts remain because the
API has no account-delete endpoint. No credentials or cookie values are printed.
"""

import http.cookiejar
import http.cookies
import json
import math
import os
import secrets
import sys
import urllib.error
import urllib.parse
import urllib.request
import uuid


BASE_URL = os.environ.get("BASE_URL", "http://localhost:8080").rstrip("/")
CSRF_COOKIE = os.environ.get("CSRF_COOKIE_NAME", "lineup_lab_csrf")
SIMULATION_FIELDS = (
    "name", "at_bat", "hit", "double", "triple", "home_run",
    "ball_on_base", "hit_by_pitch",
)


def require(condition, message):
    if not condition:
        raise AssertionError(message)


def normalized_cookie(value):
    parsed = http.cookies.SimpleCookie()
    parsed.load("token=" + value)
    require("token" in parsed, "CSRF cookie must parse")
    return parsed["token"].value


class Client:
    def __init__(self):
        self.jar = http.cookiejar.CookieJar()
        self.opener = urllib.request.build_opener(
            urllib.request.HTTPCookieProcessor(self.jar)
        )

    def csrf(self):
        values = [cookie.value for cookie in self.jar if cookie.name == CSRF_COOKIE]
        require(len(values) == 1, "login must set one CSRF cookie")
        token = normalized_cookie(values[0])
        require(len(token.split(":")) == 3, "CSRF token must use the signed session format")
        require(normalized_cookie(json.dumps(token)) == token,
                "quoted signed CSRF cookie must normalize to its unquoted header value")
        return token

    def quoted_cookie_header(self):
        token = self.csrf()
        return "; ".join(
            cookie.name + "=" + (json.dumps(token) if cookie.name == CSRF_COOKIE else cookie.value)
            for cookie in self.jar
        )

    def request(self, method, path, expected, body=None, csrf=False, headers=None):
        request_headers = {"Accept": "application/json"}
        if body is not None:
            request_headers["Content-Type"] = "application/json"
        if csrf:
            request_headers["X-CSRF-Token"] = self.csrf()
        request_headers.update(headers or {})
        request = urllib.request.Request(
            BASE_URL + path,
            data=None if body is None else json.dumps(body).encode("utf-8"),
            headers=request_headers,
            method=method,
        )
        try:
            response = self.opener.open(request, timeout=90)
        except urllib.error.HTTPError as error:
            response = error
        with response:
            status = response.code
            payload = response.read()
        require(status == expected, f"{method} {path.split('?')[0]}: expected {expected}, got {status}")
        if not payload:
            return None
        if expected >= 400:
            return None
        try:
            return json.loads(payload)
        except (ValueError, UnicodeDecodeError):
            raise AssertionError(f"{method} {path.split('?')[0]}: expected JSON response") from None

    def account(self, suffix):
        unique = uuid.uuid4().hex
        username = "smoke_" + suffix + "_" + unique[:20]
        password = "MVP smoke password " + secrets.token_urlsafe(24)
        created = self.request("POST", "/api/auth/register", 201, {
            "username": username,
            "email": username + "@example.com",
            "password": password,
        })
        logged_in = self.request("POST", "/api/auth/login", 200, {
            "username_or_email": username, "password": password,
        })
        require(logged_in["id"] == created["id"], "login must return the registered account")
        current = self.request("GET", "/api/users/me", 200)
        require(current["id"] == created["id"], "session cookie must restore the account")
        self.csrf()
        return current["id"]


def simulate(client, batters):
    payload = [{key: batter[key] for key in SIMULATION_FIELDS} for batter in batters]
    result = client.request("POST", "/api/simulate", 200, payload)
    for key in ("average_score", "average_hits"):
        value = result.get(key)
        require(isinstance(value, (int, float)) and not isinstance(value, bool)
                and math.isfinite(value) and value >= 0,
                "simulation must return finite non-negative averages")


def main():
    owner, other, anonymous = Client(), Client(), Client()
    created_ids = set()
    try:
        owner_id = owner.account("a")
        other_id = other.account("b")
        require(owner_id != other_id, "test accounts must be distinct")
        print("PASS: registration, login, signed cookies, and session restoration")

        anonymous.request("GET", "/api/lineups", 401,
                          headers={"X-Authenticated-User-ID": str(owner_id)})
        teams = owner.request("GET", "/api/teams", 200)
        require(isinstance(teams, list), "teams response must be an array")
        team = next((item for item in teams
                     if item.get("name") == "Demo Comets (Synthetic)" and item.get("year") == 2024), None)
        require(team is not None, "demo team/year must be seeded before this test")
        query = urllib.parse.urlencode({"team": team["name"], "year": team["year"]})
        roster = owner.request("GET", "/api/batting?" + query, 200)
        require(isinstance(roster, list) and len(roster) >= 9, "demo roster must contain at least nine hitters")
        selected = roster[:9]
        batter_ids = [batter["id"] for batter in selected]
        require(len(set(batter_ids)) == 9, "selected hitters must have unique stored IDs")
        simulate(owner, selected)
        print("PASS: demo roster, live IDs, and strict eight-field simulation payload")

        lineup_input = {
            "name": "MVP smoke " + uuid.uuid4().hex,
            "team": team["name"], "year": team["year"], "batter_ids": batter_ids,
        }
        owner.request("POST", "/api/lineups", 403, lineup_input)
        owner.request("POST", "/api/lineups", 403, lineup_input,
                      headers={"X-Original-Method": "GET"})
        owner.request("POST", "/api/lineups", 403, lineup_input,
                      headers={"X-CSRF-Token": other.csrf()})
        owner.request("GET", "/_verify_lineups_session", 404)
        owner.request("GET", "/api/internal/auth/verify", 404)
        anonymous.request("GET", "/api/simulations", 401)
        valid_token = owner.csrf()
        invalid_token = valid_token[:-1] + ("a" if valid_token[-1] != "a" else "b")
        invalid_cookie = owner.quoted_cookie_header().replace(json.dumps(valid_token), json.dumps(invalid_token))
        owner.request("POST", "/api/lineups", 403, lineup_input,
                      headers={"Cookie": invalid_cookie, "X-CSRF-Token": invalid_token})
        saved = owner.request("POST", "/api/lineups", 201, lineup_input, csrf=True,
                              headers={"Cookie": owner.quoted_cookie_header(),
                                       "X-Authenticated-User-ID": str(other_id)})
        lineup_id = saved["id"]
        created_ids.add(lineup_id)
        path = "/api/lineups/" + str(lineup_id)
        require(saved["batter_ids"] == batter_ids, "created lineup must preserve selected order")
        require(saved["revision"] >= 1, "created lineup must return a revision")
        require([batter["id"] for batter in saved["batters"]] == batter_ids,
                "snapshot must preserve player IDs and batting order")
        listed = owner.request("GET", "/api/lineups", 200)
        require(any(item["id"] == lineup_id for item in listed), "created lineup must appear in owner's list")
        loaded = owner.request("GET", path, 200,
                               headers={"X-Authenticated-User-ID": str(other_id)})
        require(loaded["name"] == lineup_input["name"] and loaded["batters"] == saved["batters"]
                and loaded["batter_ids"] == batter_ids, "GET must restore persisted name/order/snapshot")
        simulate(owner, loaded["batters"])
        print("PASS: CSRF enforcement, quoted-cookie normalization, persistence, and header replacement")

        update_input = dict(lineup_input, name=lineup_input["name"] + " updated",
                            batter_ids=list(reversed(batter_ids)), revision=loaded["revision"])
        forged_owner = {"X-Authenticated-User-ID": str(owner_id)}
        other.request("GET", path, 404, headers=forged_owner)
        other.request("PUT", path, 404, update_input, csrf=True, headers=forged_owner)
        other.request("DELETE", path, 404, csrf=True, headers=forged_owner)
        other_list = other.request("GET", "/api/lineups", 200, headers=forged_owner)
        require(all(item["id"] != lineup_id for item in other_list), "other owner's list must hide this lineup")
        owner.request("PUT", path, 403, update_input)
        updated = owner.request("PUT", path, 200, update_input, csrf=True)
        require(updated["revision"] == loaded["revision"] + 1, "successful update must increment revision")
        require(updated["batter_ids"] == update_input["batter_ids"]
                and [batter["id"] for batter in updated["batters"]] == update_input["batter_ids"],
                "update must persist the new batting order and snapshot")
        owner.request("PUT", path, 409, update_input, csrf=True)
        persisted = owner.request("GET", path, 200)
        require(persisted["revision"] == updated["revision"]
                and persisted["batter_ids"] == updated["batter_ids"]
                and persisted["name"] == updated["name"], "stale update must preserve latest values")
        print("PASS: cross-owner GET/PUT/DELETE isolation and stale-revision conflict")

        owner.request("DELETE", path, 403)
        owner.request("DELETE", path, 204, csrf=True)
        created_ids.remove(lineup_id)
        owner.request("GET", path, 404)
        revoked_cookie = owner.quoted_cookie_header()
        owner.request("POST", "/api/auth/logout", 200, csrf=True)
        owner.request("GET", "/api/users/me", 401)
        owner.request("GET", path, 401, headers={"Cookie": revoked_cookie,
                                                  "X-Authenticated-User-ID": str(owner_id)})
        other.request("POST", "/api/auth/logout", 200, csrf=True)
        print("PASS: owner deletion, session revocation, and logged-out resource protection")
        print("MVP HTTP smoke passed. Only uniquely generated test accounts remain.")
    finally:
        for lineup_id in created_ids:
            try:
                owner.request("DELETE", "/api/lineups/" + str(lineup_id), 204, csrf=True)
            except Exception:
                try:
                    other.request("DELETE", "/api/lineups/" + str(lineup_id), 204, csrf=True)
                except Exception:
                    print("Cleanup could not delete a test-created lineup; inspect the test accounts.", file=sys.stderr)


if __name__ == "__main__":
    try:
        main()
    except (AssertionError, KeyError, TypeError, ValueError, urllib.error.URLError, TimeoutError) as error:
        print("MVP HTTP smoke failed: " + str(error), file=sys.stderr)
        sys.exit(1)
