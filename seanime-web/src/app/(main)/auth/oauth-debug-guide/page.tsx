"use client"
import React from "react"

export default function OAuthDebugGuide() {
  return (
    <div style={{ maxWidth: 700, margin: "3rem auto", padding: 24, background: "#181A20", borderRadius: 16, color: "#fff", fontFamily: "sans-serif" }}>
      <h1 style={{ fontSize: 32, fontWeight: 700, marginBottom: 16 }}>AniList OAuth Debug & Fix Guide</h1>
      <ol style={{ fontSize: 18, lineHeight: 1.6 }}>
        <li style={{ marginBottom: 16 }}>
          <b>Check AniList Developer Settings:</b><br/>
          Go to <a href="https://anilist.co/settings/developer" target="_blank" rel="noopener noreferrer" style={{ color: "#7aa2f7" }}>AniList Developer Settings</a>.<br/>
          Make sure your <b>redirect URI</b> is set to:<br/>
          <code style={{ background: "#23272e", padding: "2px 6px", borderRadius: 4 }}>http://localhost:43211/auth/callback</code>
        </li>
        <li style={{ marginBottom: 16 }}>
          <b>Check OAuth URL in your app:</b><br/>
          It should look like:<br/>
          <code style={{ background: "#23272e", padding: "2px 6px", borderRadius: 4 }}>https://anilist.co/api/v2/oauth/authorize?client_id=26797&amp;response_type=code&amp;redirect_uri=http%3A%2F%2Flocalhost%3A43211%2Fauth%2Fcallback</code>
        </li>
        <li style={{ marginBottom: 16 }}>
          <b>Test the Callback Route:</b><br/>
          Open <code style={{ background: "#23272e", padding: "2px 6px", borderRadius: 4 }}>http://localhost:43211/auth/callback</code> in your browser.<br/>
          You should see your callback page, <b>not</b> a login screen or 404.<br/>
          <span style={{ color: "#f7768e" }}>If you see login/404, remove any AuthGuard or protection from this route.</span>
        </li>
        <li style={{ marginBottom: 16 }}>
          <b>Try the Login Flow:</b><br/>
          Click <b>Login with AniList</b>.<br/>
          Authorize the app in the popup.<br/>
          After clicking Authorize, the popup should redirect to your callback URL with <code>?code=...</code> in the URL.
        </li>
        <li style={{ marginBottom: 16 }}>
          <b>Check the Network Tab:</b><br/>
          Open browser dev tools → Network tab.<br/>
          After authorizing, look for a request to <code>/api/v1/auth/anilist/callback</code>.<br/>
          <b>You want to see:</b> <code>{`{"access_token": "..."}`}</code> in the Response.<br/>
          <span style={{ color: "#f7768e" }}>If you see an error or empty object, copy it and share it for help.</span>
        </li>
        <li style={{ marginBottom: 16 }}>
          <b>If You Get Stuck:</b><br/>
          Double-check that <b>/auth/callback</b> is NOT protected by AuthGuard.<br/>
          Make sure the redirect URI in AniList and your app <b>match exactly</b>.<br/>
          If you see errors in the network response, copy them here for help.
        </li>
      </ol>
      <hr style={{ borderColor: "#23272e", margin: "2rem 0" }} />
      <h2 style={{ fontSize: 22, marginBottom: 12 }}>Quick Checklist</h2>
      <table style={{ width: "100%", background: "#23272e", borderRadius: 8, padding: 8, color: "#fff", fontSize: 16 }}>
        <thead>
          <tr>
            <th style={{ textAlign: "left", padding: 6 }}>Step</th>
            <th style={{ textAlign: "left", padding: 6 }}>What to Check/Do</th>
            <th style={{ textAlign: "left", padding: 6 }}>What You Should See</th>
          </tr>
        </thead>
        <tbody>
          <tr>
            <td style={{ padding: 6 }}>1</td>
            <td style={{ padding: 6 }}>AniList redirect URI matches your app</td>
            <td style={{ padding: 6 }}>Exact match, e.g., /auth/callback</td>
          </tr>
          <tr>
            <td style={{ padding: 6 }}>2</td>
            <td style={{ padding: 6 }}>OAuth URL in your frontend</td>
            <td style={{ padding: 6 }}>Correct redirect_uri parameter</td>
          </tr>
          <tr>
            <td style={{ padding: 6 }}>3</td>
            <td style={{ padding: 6 }}>Direct visit /auth/callback in browser</td>
            <td style={{ padding: 6 }}>Your callback page, not login/404</td>
          </tr>
          <tr>
            <td style={{ padding: 6 }}>4</td>
            <td style={{ padding: 6 }}>Login flow after "Authorize"</td>
            <td style={{ padding: 6 }}>Redirects to /auth/callback?code=...</td>
          </tr>
          <tr>
            <td style={{ padding: 6 }}>5</td>
            <td style={{ padding: 6 }}>Network tab after login</td>
            <td style={{ padding: 6 }}>POST /api/v1/auth/anilist/callback → {`{"access_token": "..."}`}</td>
          </tr>
        </tbody>
      </table>
      <p style={{ marginTop: 24, color: "#7aa2f7" }}>
        <b>If any step fails, copy what you see and share it for direct help!</b>
      </p>
    </div>
  )
}
