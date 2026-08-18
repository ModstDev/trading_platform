const DEFAULT_API="http://localhost:8080";
const $=id=>document.getElementById(id);
function base(){return $("apiBase").value.replace(/\/+$/,"")}
function token(){return localStorage.getItem("access_token")}
function status(){ $("authStatus").textContent=token()?"Logged in":"Not logged in" }
function output(x){$("output").textContent=typeof x==="string"?x:JSON.stringify(x,null,2)}

async function request(path,options={}){
  const headers=options.headers||{};
  if(options.body) headers["Content-Type"]="application/json";
  if(token()) headers.Authorization=`Bearer ${token()}`;
  const r=await fetch(base()+path,{...options,headers});
  const text=await r.text();
  let data; try{data=text?JSON.parse(text):null}catch{data=text}
  if(!r.ok) throw new Error(`${r.status}: ${typeof data==="string"?data:JSON.stringify(data,null,2)}`);
  return data;
}

async function register(){
  try{
    const data=await request("/users",{method:"POST",body:JSON.stringify({
      email:$("registerEmail").value,password:$("registerPassword").value})});
    output(data);
  }catch(e){output(e.message)}
}
async function login(){
  try{
    const data=await request("/login",{method:"POST",body:JSON.stringify({
      email:$("loginEmail").value,password:$("loginPassword").value})});
    if(!data.access_token) throw new Error("No access_token returned");
    localStorage.setItem("access_token",data.access_token); status(); output(data);
    await loadAccount(); await loadOrders();
  }catch(e){output(e.message)}
}
function logout(){localStorage.removeItem("access_token");status();output("Logged out.")}
async function loadAccount(){
  try{$("accountOutput").textContent=JSON.stringify(await request("/account"),null,2)}
  catch(e){$("accountOutput").textContent=e.message}
}
async function loadInstruments(){
  try{
    const data=await request("/instruments");
    $("instrumentsOutput").textContent=JSON.stringify(data,null,2);
    if(Array.isArray(data)&&data[0]?.id) $("instrumentId").value=data[0].id;
  }catch(e){$("instrumentsOutput").textContent=e.message}
}
async function createOrder(){
  try{
    const type=$("orderType").value;
    const body={instrument_id:$("instrumentId").value,side:$("side").value,type,
      quantity:$("quantity").value};
    if(type==="LIMIT") body.price=$("price").value;
    const data=await request("/orders",{method:"POST",body:JSON.stringify(body)});
    $("orderOutput").textContent=JSON.stringify(data,null,2);output(data);
    await loadOrders(); await loadAccount();
  }catch(e){$("orderOutput").textContent=e.message;output(e.message)}
}
async function loadOrders() {
  try {
    const data = await request("/orders");
    const box = $("orders");

    box.innerHTML = "";

    if (!Array.isArray(data) || !data.length) {
      box.textContent = "No orders.";
      return;
    }

    for (const o of data) {
      const id = o.id ?? o.ID;
      const status = o.status ?? o.Status;
      const quantity = o.quantity ?? o.Quantity;
      const filledQuantity = o.filled_quantity ?? o.FilledQuantity;
      const side = o.side ?? o.Side;
      const type = o.type ?? o.Type;

      let price = o.price ?? o.Price;

      // Handle sql.NullString / similar JSON object.
      if (price && typeof price === "object") {
        if ("String" in price) {
          price = price.String;
        } else if ("string" in price) {
          price = price.string;
        } else if ("Valid" in price && !price.Valid) {
          price = "—";
        } else {
          price = JSON.stringify(price);
        }
      }

      const div = document.createElement("div");
      div.className = "order";

      div.innerHTML = `
        <b>${side} ${type}</b><br>
        ID: ${id}<br>
        Quantity: ${quantity}<br>
        Filled: ${filledQuantity}<br>
        Price: ${price ?? "—"}<br>
        Status: ${status}<br>

        <button
          class="danger"
          onclick="cancelOrder('${id}')">
          Cancel
        </button>
      `;

      box.appendChild(div);
    }
  } catch (e) {
    $("orders").textContent = e.message;
  }
}
async function cancelOrder(id){
  if(!confirm("Cancel order?"))return;
  try{output(await request(`/orders/${id}`,{method:"DELETE"}));await loadOrders();await loadAccount()}
  catch(e){output(e.message)}
}
async function loadPositions(){
  try{$("positionsOutput").textContent=JSON.stringify(await request("/positions"),null,2)}
  catch(e){$("positionsOutput").textContent=e.message}
}
async function loadExecutions(){
  try{$("executionsOutput").textContent=JSON.stringify(await request("/executions"),null,2)}
  catch(e){$("executionsOutput").textContent=e.message}
}
$("apiBase").value=localStorage.getItem("api_base")||DEFAULT_API;
$("apiBase").addEventListener("change",()=>localStorage.setItem("api_base",base()));
status();