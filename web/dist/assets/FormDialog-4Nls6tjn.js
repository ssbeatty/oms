import{w as j,r as m,o as u,_ as D,q as T,j as e}from"./index-vyLcd-Je.js";import{D as x,a as g,b,c as p}from"./DialogActions-TfqgKDOq.js";import{T as C,B as c}from"./Button-P3xriBeC.js";var v={root:{margin:0,padding:"16px 24px",flex:"0 0 auto"}},k=m.forwardRef(function(a,l){var i=a.children,o=a.classes,s=a.className,r=a.disableTypography,t=r===void 0?!1:r,d=u(a,["children","classes","className","disableTypography"]);return m.createElement("div",D({className:T(o.root,s),ref:l},d),t?i:m.createElement(C,{component:"h2",variant:"h6"},i))});const f=j(v,{name:"MuiDialogTitle"})(k);function _({open:n=!1,title:a="",text:l="",toClose:i,todo:o}){const s=()=>{i&&i()},r=()=>{o&&o(),i()};return e.jsxs(x,{open:n,onClose:s,"aria-labelledby":"is-delete-host",children:[e.jsx(f,{style:{backgroundColor:"#ecad5a"},children:a}),e.jsx(g,{dividers:!0,children:e.jsx(b,{children:l})}),e.jsxs(p,{children:[e.jsx(c,{onClick:s,color:"primary",children:"取消"}),e.jsx(c,{onClick:r,color:"primary",children:"确定"})]})]})}function B(n){const{open:a=!1,title:l,content:i,toClose:o,todo:s,maxWidth:r}=n,t=(h,y)=>{y!=="backdropClick"&&o&&o()},d=async()=>{await s()&&o()};return e.jsxs(x,{open:a,title:l,onClose:t,maxWidth:r,"aria-labelledby":"form-dialog",children:[e.jsx(f,{id:"form-dialog",children:l}),e.jsx(g,{children:i||null}),e.jsxs(p,{children:[e.jsx(c,{onClick:h=>t(h,""),color:"primary",children:"取消"}),e.jsx(c,{onClick:d,color:"primary",children:"确认"})]})]})}export{f as D,B as F,_ as T};
