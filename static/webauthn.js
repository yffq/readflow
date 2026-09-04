(function () {
  'use strict';

  function b64urlToBuf(b64url) {
    var pad = '='.repeat((4 - (b64url.length % 4)) % 4);
    var b64 = (b64url + pad).replace(/-/g, '+').replace(/_/g, '/');
    var raw = atob(b64);
    var buf = new Uint8Array(raw.length);
    for (var i = 0; i < raw.length; i++) buf[i] = raw.charCodeAt(i);
    return buf.buffer;
  }

  function bufToB64url(buf) {
    var bytes = new Uint8Array(buf);
    var binary = '';
    for (var i = 0; i < bytes.length; i++) binary += String.fromCharCode(bytes[i]);
    return btoa(binary).replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/, '');
  }

  function postJSON(url, data) {
    return fetch(url, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', 'Accept': 'application/json' },
      body: JSON.stringify(data || {})
    }).then(function (res) {
      return res.json().catch(function () { return {}; }).then(function (body) {
        return { ok: res.ok, status: res.status, body: body };
      });
    });
  }

  function registrationOptions(server) {
    var pk = server.publicKey;
    var out = {
      challenge: b64urlToBuf(pk.challenge),
      rp: pk.rp,
      user: {
        id: b64urlToBuf(pk.user.id),
        name: pk.user.name,
        displayName: pk.user.displayName
      },
      pubKeyCredParams: pk.pubKeyCredParams,
      timeout: pk.timeout,
      attestation: pk.attestation
    };
    if (pk.authenticatorSelection) out.authenticatorSelection = pk.authenticatorSelection;
    if (pk.excludeCredentials && pk.excludeCredentials.length) {
      out.excludeCredentials = pk.excludeCredentials.map(function (c) {
        return { type: c.type, id: b64urlToBuf(c.id), transports: c.transports };
      });
    }
    return out;
  }

  function assertionOptions(server) {
    var pk = server.publicKey;
    var out = {
      challenge: b64urlToBuf(pk.challenge),
      rpId: pk.rpId,
      timeout: pk.timeout,
      userVerification: pk.userVerification
    };
    if (pk.allowCredentials && pk.allowCredentials.length) {
      out.allowCredentials = pk.allowCredentials.map(function (c) {
        return { type: c.type, id: b64urlToBuf(c.id), transports: c.transports };
      });
    }
    return out;
  }

  function creationToJSON(cred) {
    return {
      id: cred.id,
      rawId: bufToB64url(cred.rawId),
      type: cred.type,
      response: {
        clientDataJSON: bufToB64url(cred.response.clientDataJSON),
        attestationObject: bufToB64url(cred.response.attestationObject)
      }
    };
  }

  function assertionToJSON(cred) {
    return {
      id: cred.id,
      rawId: bufToB64url(cred.rawId),
      type: cred.type,
      response: {
        clientDataJSON: bufToB64url(cred.response.clientDataJSON),
        authenticatorData: bufToB64url(cred.response.authenticatorData),
        signature: bufToB64url(cred.response.signature),
        userHandle: cred.response.userHandle ? bufToB64url(cred.response.userHandle) : null
      }
    };
  }

  function fail(msg) {
    alert(msg || '操作失败');
  }

  window.loginWithPasskey = function () {
    if (!window.PublicKeyCredential) {
      fail('当前浏览器不支持指纹登录（WebAuthn）');
      return;
    }
    postJSON('/webauthn/login/begin', {})
      .then(function (res) {
        if (!res.ok) throw new Error(res.body.error || '无法开始指纹登录');
        return navigator.credentials.get({ publicKey: assertionOptions(res.body) });
      })
      .then(function (cred) {
        return postJSON('/webauthn/login/finish', assertionToJSON(cred));
      })
      .then(function (res) {
        if (!res.ok) throw new Error(res.body.error || '指纹验证失败');
        window.location.href = '/';
      })
      .catch(function (err) {
        fail('指纹登录失败：' + (err.message || err));
      });
  };

  window.registerPasskey = function () {
    if (!window.PublicKeyCredential) {
      fail('当前浏览器不支持指纹登录（WebAuthn）');
      return;
    }
    postJSON('/webauthn/register/begin', {})
      .then(function (res) {
        if (!res.ok) throw new Error(res.body.error || '无法开始注册');
        return navigator.credentials.create({ publicKey: registrationOptions(res.body) });
      })
      .then(function (cred) {
        return postJSON('/webauthn/register/finish', creationToJSON(cred));
      })
      .then(function (res) {
        if (!res.ok) throw new Error(res.body.error || '注册失败');
        alert('指纹登录已添加');
        window.location.reload();
      })
      .catch(function (err) {
        fail('添加失败：' + (err.message || err));
      });
  };
})();
