/*
Licensed to the Apache Software Foundation (ASF) under one
or more contributor license agreements.  See the NOTICE file
distributed with this work for additional information
regarding copyright ownership.  The ASF licenses this file
to you under the Apache License, Version 2.0 (the
"License"); you may not use this file except in compliance
with the License.  You may obtain a copy of the License at

  http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing,
software distributed under the License is distributed on an
"AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
KIND, either express or implied.  See the License for the
specific language governing permissions and limitations
under the License.
*/

/* AMCL BLS Curve pairing functions */

//#define HAS_MAIN

#include "pair192_ZZZ.h"

using namespace XXX;
using namespace YYY;

namespace ZZZ {
	static void PAIR_line(FP24 *,ECP4 *,ECP4 *,FP *,FP *);
	static void glv(BIG u[2],BIG);
	static void gs(BIG u[4],BIG);
}

/* Line function */
static void ZZZ::PAIR_line(FP24 *v,ECP4 *A,ECP4 *B,FP *Qx,FP *Qy)
{
	FP4 X1,Y1,T1,T2;
	FP4 XX,YY,ZZ,YZ;
    FP8 a,b,c;

	if (A==B)
    {
        /* doubling */
 		FP4_copy(&XX,&(A->x));	//FP4 XX=new FP4(A.getx());  //X
		FP4_copy(&YY,&(A->y));	//FP4 YY=new FP4(A.gety());  //Y
		FP4_copy(&ZZ,&(A->z));	//FP4 ZZ=new FP4(A.getz());  //Z


		FP4_copy(&YZ,&YY);		//FP4 YZ=new FP4(YY);        //Y 
		FP4_mul(&YZ,&YZ,&ZZ);		//YZ.mul(ZZ);                //YZ
		FP4_sqr(&XX,&XX);		//XX.sqr();	               //X^2
		FP4_sqr(&YY,&YY);		//YY.sqr();	               //Y^2
		FP4_sqr(&ZZ,&ZZ);		//ZZ.sqr();			       //Z^2
			
		FP4_imul(&YZ,&YZ,4);	//YZ.imul(4);
		FP4_neg(&YZ,&YZ);		//YZ.neg(); 
		FP4_norm(&YZ);			//YZ.norm();       //-4YZ

		FP4_imul(&XX,&XX,6);					//6X^2
		FP4_qmul(&XX,&XX,Qx);	               //6X^2.Xs

		FP4_imul(&ZZ,&ZZ,3*CURVE_B_I);	//3Bz^2 

		FP4_qmul(&YZ,&YZ,Qy);	//-4YZ.Ys

#if SEXTIC_TWIST_ZZZ==D_TYPE
		FP4_div_2i(&ZZ);		//6(b/i)z^2
#endif
#if SEXTIC_TWIST_ZZZ==M_TYPE
		FP4_times_i(&ZZ);
		FP4_add(&ZZ,&ZZ,&ZZ);  // 6biz^2
		FP4_times_i(&YZ);
		FP4_norm(&YZ);	
#endif
		FP4_norm(&ZZ);			// 6bi.Z^2 

		FP4_add(&YY,&YY,&YY);	// 2y^2
		FP4_sub(&ZZ,&ZZ,&YY);	// 
		FP4_norm(&ZZ);			// 6b.Z^2-2Y^2

		FP8_from_FP4s(&a,&YZ,&ZZ); // -4YZ.Ys | 6b.Z^2-2Y^2 | 6X^2.Xs 
#if SEXTIC_TWIST_ZZZ==D_TYPE
		FP8_from_FP4(&b,&XX);	
		FP8_zero(&c);
#endif
#if SEXTIC_TWIST_ZZZ==M_TYPE
		FP8_zero(&b);
		FP8_from_FP4H(&c,&XX);
#endif

		ECP4_dbl(A);				//A.dbl();
    }
    else
    {
        /* addition */

		FP4_copy(&X1,&(A->x));		//FP4 X1=new FP4(A.getx());    // X1
		FP4_copy(&Y1,&(A->y));		//FP4 Y1=new FP4(A.gety());    // Y1
		FP4_copy(&T1,&(A->z));		//FP4 T1=new FP4(A.getz());    // Z1
			
		FP4_copy(&T2,&T1);		//FP4 T2=new FP4(A.getz());    // Z1

		FP4_mul(&T1,&T1,&(B->y));	//T1.mul(B.gety());    // T1=Z1.Y2 
		FP4_mul(&T2,&T2,&(B->x));	//T2.mul(B.getx());    // T2=Z1.X2

		FP4_sub(&X1,&X1,&T2);		//X1.sub(T2); 
		FP4_norm(&X1);				//X1.norm();  // X1=X1-Z1.X2
		FP4_sub(&Y1,&Y1,&T1);		//Y1.sub(T1); 
		FP4_norm(&Y1);				//Y1.norm();  // Y1=Y1-Z1.Y2

		FP4_copy(&T1,&X1);			//T1.copy(X1);            // T1=X1-Z1.X2

		FP4_qmul(&X1,&X1,Qy);		//X1.pmul(Qy);            // X1=(X1-Z1.X2).Ys
#if SEXTIC_TWIST_ZZZ==M_TYPE
		FP4_times_i(&X1);
		FP4_norm(&X1);
#endif

		FP4_mul(&T1,&T1,&(B->y));	//T1.mul(B.gety());       // T1=(X1-Z1.X2).Y2

		FP4_copy(&T2,&Y1);			//T2.copy(Y1);            // T2=Y1-Z1.Y2
		FP4_mul(&T2,&T2,&(B->x));	//T2.mul(B.getx());       // T2=(Y1-Z1.Y2).X2
		FP4_sub(&T2,&T2,&T1);		//T2.sub(T1); 
		FP4_norm(&T2);				//T2.norm();          // T2=(Y1-Z1.Y2).X2 - (X1-Z1.X2).Y2
		FP4_qmul(&Y1,&Y1,Qx);		//Y1.pmul(Qx);  
		FP4_neg(&Y1,&Y1);			//Y1.neg(); 
		FP4_norm(&Y1);				//Y1.norm(); // Y1=-(Y1-Z1.Y2).Xs

		FP8_from_FP4s(&a,&X1,&T2);	// (X1-Z1.X2).Ys  |  (Y1-Z1.Y2).X2 - (X1-Z1.X2).Y2  | - (Y1-Z1.Y2).Xs
#if SEXTIC_TWIST_ZZZ==D_TYPE
		FP8_from_FP4(&b,&Y1);		//b=new FP4(Y1);
		FP8_zero(&c);
#endif
#if SEXTIC_TWIST_ZZZ==M_TYPE
		FP8_zero(&b);
		FP8_from_FP4H(&c,&Y1);		//b=new FP4(Y1);
#endif
		ECP4_add(A,B);			//A.add(B);
    }

    FP24_from_FP8s(v,&a,&b,&c);
}

/* Optimal R-ate pairing r=e(P,Q) */
void ZZZ::PAIR_ate(FP24 *r,ECP4 *P1,ECP *Q1)
{
    BIG x,n,n3;
	FP Qx,Qy;
    int i,j,nb,bt;
    ECP4 A,NP,P;
	ECP Q;
    FP24 lv;

    BIG_rcopy(x,CURVE_Bnx);

    BIG_copy(n,x);

	BIG_pmul(n3,n,3);
	BIG_norm(n3);

	ECP4_copy(&P,P1);
	ECP_copy(&Q,Q1);

	ECP4_affine(&P);
	ECP_affine(&Q);

    FP_copy(&Qx,&(Q.x));
    FP_copy(&Qy,&(Q.y));

    ECP4_copy(&A,&P);

	ECP4_copy(&NP,&P); ECP4_neg(&NP);

    FP24_one(r);
    nb=BIG_nbits(n3);  // n3

	j=0;
    /* Main Miller Loop */
    for (i=nb-2; i>=1; i--)
    {
		j++;
		FP24_sqr(r,r);
        PAIR_line(&lv,&A,&A,&Qx,&Qy);
        FP24_smul(r,&lv,SEXTIC_TWIST_ZZZ);

		bt= BIG_bit(n3,i)-BIG_bit(n,i);  // BIG_bit(n,i); 
        if (bt==1)
        {
            PAIR_line(&lv,&A,&P,&Qx,&Qy);
            FP24_smul(r,&lv,SEXTIC_TWIST_ZZZ);
        }
		if (bt==-1)
		{
            PAIR_line(&lv,&A,&NP,&Qx,&Qy);
            FP24_smul(r,&lv,SEXTIC_TWIST_ZZZ);
		}

    }

#if SIGN_OF_X_ZZZ==NEGATIVEX
    FP24_conj(r,r);
#endif

}

/* Optimal R-ate double pairing e(P,Q).e(R,S) */
void ZZZ::PAIR_double_ate(FP24 *r,ECP4 *P1,ECP *Q1,ECP4 *R1,ECP *S1)
{
    BIG x,n,n3;
	FP Qx,Qy,Sx,Sy;
    int i,nb,bt;
    ECP4 A,B,NP,NR,P,R;
	ECP Q,S;
    FP24 lv;

    BIG_rcopy(x,CURVE_Bnx);
    BIG_copy(n,x);

	BIG_pmul(n3,n,3);
	BIG_norm(n3);

	ECP4_copy(&P,P1);
	ECP_copy(&Q,Q1);

	ECP4_affine(&P);
	ECP_affine(&Q);

	ECP4_copy(&R,R1);
	ECP_copy(&S,S1);

	ECP4_affine(&R);
	ECP_affine(&S);

    FP_copy(&Qx,&(Q.x));
    FP_copy(&Qy,&(Q.y));

    FP_copy(&Sx,&(S.x));
    FP_copy(&Sy,&(S.y));

    ECP4_copy(&A,&P);
    ECP4_copy(&B,&R);

	ECP4_copy(&NP,&P); ECP4_neg(&NP);
	ECP4_copy(&NR,&R); ECP4_neg(&NR);


    FP24_one(r);
    nb=BIG_nbits(n3);

    /* Main Miller Loop */
    for (i=nb-2; i>=1; i--)
    {
        FP24_sqr(r,r);
        PAIR_line(&lv,&A,&A,&Qx,&Qy);
        FP24_smul(r,&lv,SEXTIC_TWIST_ZZZ);

        PAIR_line(&lv,&B,&B,&Sx,&Sy);
        FP24_smul(r,&lv,SEXTIC_TWIST_ZZZ);

		bt=BIG_bit(n3,i)-BIG_bit(n,i); // bt=BIG_bit(n,i);
        if (bt==1)
        {
            PAIR_line(&lv,&A,&P,&Qx,&Qy);
            FP24_smul(r,&lv,SEXTIC_TWIST_ZZZ);

            PAIR_line(&lv,&B,&R,&Sx,&Sy);
            FP24_smul(r,&lv,SEXTIC_TWIST_ZZZ);
        }
		if (bt==-1)
		{
            PAIR_line(&lv,&A,&NP,&Qx,&Qy);
            FP24_smul(r,&lv,SEXTIC_TWIST_ZZZ);
            PAIR_line(&lv,&B,&NR,&Sx,&Sy);
            FP24_smul(r,&lv,SEXTIC_TWIST_ZZZ);
		}
	}



#if SIGN_OF_X_ZZZ==NEGATIVEX
    FP24_conj(r,r);
#endif

}

/* final exponentiation - keep separate for multi-pairings and to avoid thrashing stack */

void ZZZ::PAIR_fexp(FP24 *r)
{
    FP2 X;
    BIG x;
	FP a,b;
    FP24 t0,t1,t2,t3,t4,t5,t6,t7;  // could lose one of these - r=t3

    BIG_rcopy(x,CURVE_Bnx);
    FP_rcopy(&a,Fra);
    FP_rcopy(&b,Frb);
    FP2_from_FPs(&X,&a,&b);

    /* Easy part of final exp - r^(p^12-1)(p^4+1)*/

    FP24_inv(&t0,r);
    FP24_conj(r,r);

    FP24_mul(r,&t0);
    FP24_copy(&t0,r);

    FP24_frob(r,&X,4);

    FP24_mul(r,&t0);

// Ghamman & Fouotsa Method - (completely garbled in  https://eprint.iacr.org/2016/130)

	FP24_usqr(&t7,r);			// t7=f^2
	FP24_pow(&t1,&t7,x);		// t1=t7^u

	BIG_fshr(x,1);
	FP24_pow(&t2,&t1,x);		// t2=t1^(u/2)
	BIG_fshl(x,1);  // x must be even

#if SIGN_OF_X_ZZZ==NEGATIVEX
	FP24_conj(&t1,&t1);
#endif

	FP24_conj(&t3,&t1);		// t3=1/t1
	FP24_mul(&t2,&t3);		// t2=t1*t3
	FP24_mul(&t2,r);		// t2=t2*f


	FP24_pow(&t3,&t2,x);		// t3=t2^u
	FP24_pow(&t4,&t3,x);		// t4=t3^u
	FP24_pow(&t5,&t4,x);		// t5=t4^u

#if SIGN_OF_X_ZZZ==NEGATIVEX
	FP24_conj(&t3,&t3);
	FP24_conj(&t5,&t5);
#endif

	FP24_frob(&t3,&X,6);
	FP24_frob(&t4,&X,5);

	FP24_mul(&t3,&t4);		// t3=t3.t4


	FP24_pow(&t6,&t5,x);		// t6=t5^u
#if SIGN_OF_X_ZZZ==NEGATIVEX
	FP24_conj(&t6,&t6);
#endif


	FP24_frob(&t5,&X,4); 
	FP24_mul(&t3,&t5); // ??


	FP24_conj(&t0,&t2);			// t0=1/t2
	FP24_mul(&t6,&t0);		// t6=t6*t0

	FP24_copy(&t5,&t6);
	FP24_frob(&t5,&X,3); 

	FP24_mul(&t3,&t5);		// t3=t3*t5
	FP24_pow(&t5,&t6,x);	// t5=t6^x
	FP24_pow(&t6,&t5,x);	// t6=t5^x

#if SIGN_OF_X_ZZZ==NEGATIVEX
	FP24_conj(&t5,&t5);
#endif

	FP24_copy(&t0,&t5);	
	FP24_frob(&t0,&X,2); 
	FP24_mul(&t3,&t0);		// t3=t3*t0
	FP24_copy(&t0,&t6);     // 
	FP24_frob(&t0,&X,1);

	FP24_mul(&t3,&t0);		// t3=t3*t0
	FP24_pow(&t5,&t6,x);    // t5=t6*x

#if SIGN_OF_X_ZZZ==NEGATIVEX
	FP24_conj(&t5,&t5);
#endif

	FP24_frob(&t2,&X,7); 

	FP24_mul(&t5,&t7);		// t5=t5*t7
	FP24_mul(&t3,&t2);		// t3=t3*t2
	FP24_mul(&t3,&t5);		// t3=t3*t5

	FP24_mul(r,&t3);
	FP24_reduce(r);

}

#ifdef USE_GLV_ZZZ
/* GLV method */
static void ZZZ::glv(BIG u[2],BIG e)
{

// -(x^4).P = (Beta.x,y)

    BIG x,x2,q;
    BIG_rcopy(x,CURVE_Bnx);
    BIG_smul(x2,x,x);
	BIG_smul(x,x2,x2);
    BIG_copy(u[0],e);
    BIG_mod(u[0],x);
    BIG_copy(u[1],e);
    BIG_sdiv(u[1],x);

    BIG_rcopy(q,CURVE_Order);
    BIG_sub(u[1],q,u[1]);


    return;
}
#endif // USE_GLV

/* Galbraith & Scott Method */
static void ZZZ::gs(BIG u[8],BIG e)
{
    int i;

    BIG x,w,q;
	BIG_rcopy(q,CURVE_Order);
    BIG_rcopy(x,CURVE_Bnx);
    BIG_copy(w,e);

    for (i=0; i<7; i++)
    {
        BIG_copy(u[i],w);
        BIG_mod(u[i],x);
        BIG_sdiv(w,x);
    }
	BIG_copy(u[7],w);

/*  */
#if SIGN_OF_X_ZZZ==NEGATIVEX
	BIG_modneg(u[1],u[1],q);
	BIG_modneg(u[3],u[3],q);
	BIG_modneg(u[5],u[5],q);
	BIG_modneg(u[7],u[7],q);
#endif


    return;
}

/* Multiply P by e in group G1 */
void ZZZ::PAIR_G1mul(ECP *P,BIG e)
{
#ifdef USE_GLV_ZZZ   /* Note this method is patented */
    int np,nn;
    ECP Q;
	FP cru;
    BIG t,q;
    BIG u[2];

    BIG_rcopy(q,CURVE_Order);
    glv(u,e);

    ECP_copy(&Q,P); ECP_affine(&Q);
    FP_rcopy(&cru,CURVE_Cru);
    FP_mul(&(Q.x),&(Q.x),&cru);

    /* note that -a.B = a.(-B). Use a or -a depending on which is smaller */

    np=BIG_nbits(u[0]);
    BIG_modneg(t,u[0],q);
    nn=BIG_nbits(t);
    if (nn<np)
    {
        BIG_copy(u[0],t);
        ECP_neg(P);
    }

    np=BIG_nbits(u[1]);
    BIG_modneg(t,u[1],q);
    nn=BIG_nbits(t);
    if (nn<np)
    {
        BIG_copy(u[1],t);
        ECP_neg(&Q);
    }
    BIG_norm(u[0]);
    BIG_norm(u[1]);    
    ECP_mul2(P,&Q,u[0],u[1]);

#else
    ECP_mul(P,e);
#endif
}

/* Multiply P by e in group G2 */
void ZZZ::PAIR_G2mul(ECP4 *P,BIG e)
{
#ifdef USE_GS_G2_ZZZ   /* Well I didn't patent it :) */
    int i,np,nn;
    ECP4 Q[8];
    FP2 X[3];
    BIG x,y,u[8];

	ECP4_frob_constants(X);

    BIG_rcopy(y,CURVE_Order);
    gs(u,e);

    ECP4_copy(&Q[0],P);
    for (i=1; i<8; i++)
    {
        ECP4_copy(&Q[i],&Q[i-1]);
        ECP4_frob(&Q[i],X,1);
    }

    for (i=0; i<8; i++)
    {
        np=BIG_nbits(u[i]);
        BIG_modneg(x,u[i],y);
        nn=BIG_nbits(x);
        if (nn<np)
        {
            BIG_copy(u[i],x);
            ECP4_neg(&Q[i]);
        }
        BIG_norm(u[i]);     
    }

    ECP4_mul8(P,Q,u);

#else
    ECP4_mul(P,e);
#endif
}

/* f=f^e */
void ZZZ::PAIR_GTpow(FP24 *f,BIG e)
{
#ifdef USE_GS_GT_ZZZ   /* Note that this option requires a lot of RAM! Maybe better to use compressed XTR method, see FP8.c */
    int i,np,nn;
    FP24 g[8];
    FP2 X;
    BIG t,q;
	FP fx,fy;
    BIG u[8];

    FP_rcopy(&fx,Fra);
    FP_rcopy(&fy,Frb);
    FP2_from_FPs(&X,&fx,&fy);

    BIG_rcopy(q,CURVE_Order);
    gs(u,e);

    FP24_copy(&g[0],f);
    for (i=1; i<8; i++)
    {
        FP24_copy(&g[i],&g[i-1]);
        FP24_frob(&g[i],&X,1);
    }

    for (i=0; i<8; i++)
    {
        np=BIG_nbits(u[i]);
        BIG_modneg(t,u[i],q);
        nn=BIG_nbits(t);
        if (nn<np)
        {
            BIG_copy(u[i],t);
            FP24_conj(&g[i],&g[i]);
        }
        BIG_norm(u[i]);        
    }
    FP24_pow8(f,g,u);

#else
    FP24_pow(f,f,e);
#endif
}


#ifdef HAS_MAIN

using namespace std;
using namespace ZZZ;


// g++ -O2 pair192_BLS24.cpp ecp4_BLS24.cpp fp24_BLS24.cpp fp8_BLS24.cpp fp4_BLS24.cpp fp2_BLS24.cpp ecp_BLS24.cpp fp_BLS24.cpp big_XXX.cpp rom_curve_BLS24.cpp rom_field_BLS24.cpp rand.cpp hash.cpp oct.cpp -o pair192_BLS24.exe

int main()
{
    int i;
    char byt[32];
    csprng rng;
    BIG xa,xb,ya,yb,w,a,b,t1,q,u[2],v[4],m,r,xx,x2,x4,p;
    ECP4 P,G;
    ECP Q,R;
    FP24 g,gp;
    FP8 t,c,cp,cpm1,cpm2;
	FP4 X,Y;
    FP2 x,y,f,Aa,Bb;
	FP cru;

	for (i=0;i<32;i++)
		byt[i]=i+9;
	RAND_seed(&rng,32,byt);

	BIG_rcopy(r,CURVE_Order);
	BIG_rcopy(p,Modulus);


    BIG_rcopy(xa,CURVE_Gx);
    BIG_rcopy(ya,CURVE_Gy);

    ECP_set(&Q,xa,ya);
    if (Q.inf) printf("Failed to set - point not on curve\n");
    else printf("G1 set success\n");

    printf("Q= ");
    ECP_output(&Q);
    printf("\n");

	ECP4_generator(&P);

    if (P.inf) printf("Failed to set - point not on curve\n");
    else printf("G2 set success\n");

    BIG_rcopy(a,Fra);
    BIG_rcopy(b,Frb);
    FP2_from_BIGs(&f,a,b);

    PAIR_ate(&g,&P,&Q);

	printf("gb= ");
    FP24_output(&g);
    printf("\n");
    PAIR_fexp(&g);

    printf("g= ");
    FP24_output(&g);
    printf("\n");

	ECP_copy(&R,&Q);
	ECP4_copy(&G,&P);

	ECP4_dbl(&G);
	ECP_dbl(&R);
	ECP_affine(&R);

    PAIR_ate(&g,&G,&Q);
    PAIR_fexp(&g);

    printf("g1= ");
    FP24_output(&g);
    printf("\n");

    PAIR_ate(&g,&P,&R);
    PAIR_fexp(&g);

    printf("g2= ");
    FP24_output(&g);
    printf("\n");


	PAIR_G1mul(&Q,r);
	printf("rQ= ");ECP_output(&Q); printf("\n");

	PAIR_G2mul(&P,r);
	printf("rP= ");ECP4_output(&P); printf("\n");

	BIG_randomnum(w,r,&rng);

	FP24_copy(&gp,&g);

	PAIR_GTpow(&g,w);

	FP24_trace(&t,&g);

	printf("g^r=  ");FP8_output(&t); printf("\n");

	FP24_compow(&t,&gp,w,r);

	printf("t(g)= "); FP8_output(&t); printf("\n");

}

#endif
